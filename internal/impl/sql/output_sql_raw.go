package sql

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/benthosdev/benthos/v4/internal/shutdown"
	"github.com/benthosdev/benthos/v4/public/bloblang"
	"github.com/benthosdev/benthos/v4/public/service"
)

func sqlRawOutputConfig() *service.ConfigSpec {
	spec := service.NewConfigSpec().
		Stable().
		Categories("Services").
		Summary("Executes an arbitrary SQL query for each message.").
		Description(``).
		Field(driverField).
		Field(dsnField).
		Field(rawQueryField().
			Example("INSERT INTO footable (foo, bar, baz) VALUES (?, ?, ?);")).
		Field(service.NewBloblangField("args_mapping").
			Description("An optional [Bloblang mapping](/docs/guides/bloblang/about) which should evaluate to an array of values matching in size to the number of placeholder arguments in the field `query`.").
			Example("root = [ this.cat.meow, this.doc.woofs[0] ]").
			Example(`root = [ meta("user.id") ]`).
			Optional()).
		Field(service.NewIntField("max_in_flight").
			Description("The maximum number of inserts to run in parallel.").
			Default(64))

	for _, f := range connFields() {
		spec = spec.Field(f)
	}

	spec = spec.Field(service.NewBatchPolicyField("batching")).
		Version("3.65.0").
		Example("Table Insert (MySQL)",
			`
Here we insert rows into a database by populating the columns id, name and topic with values extracted from messages and metadata:`,
			`
output:
  sql_raw:
    driver: mysql
    dsn: foouser:foopassword@tcp(localhost:3306)/foodb
    query: "INSERT INTO footable (id, name, topic) VALUES (?, ?, ?);"
    args_mapping: |
      root = [
        this.user.id,
        this.user.name,
        meta("kafka_topic"),
      ]
`,
		)
	return spec
}

func init() {
	err := service.RegisterBatchOutput(
		"sql_raw", sqlRawOutputConfig(),
		func(conf *service.ParsedConfig, mgr *service.Resources) (out service.BatchOutput, batchPolicy service.BatchPolicy, maxInFlight int, err error) {
			if batchPolicy, err = conf.FieldBatchPolicy("batching"); err != nil {
				return
			}
			if maxInFlight, err = conf.FieldInt("max_in_flight"); err != nil {
				return
			}
			out, err = newSQLRawOutputFromConfig(conf, mgr.Logger())
			return
		})

	if err != nil {
		panic(err)
	}
}

//------------------------------------------------------------------------------

type sqlRawOutput struct {
	driver string
	dsn    string
	db     *sql.DB
	dbMut  sync.RWMutex

	queryStatic string

	argsMapping *bloblang.Executor

	connSettings connSettings

	logger  *service.Logger
	shutSig *shutdown.Signaller
}

func newSQLRawOutputFromConfig(conf *service.ParsedConfig, logger *service.Logger) (*sqlRawOutput, error) {
	driverStr, err := conf.FieldString("driver")
	if err != nil {
		return nil, err
	}

	dsnStr, err := conf.FieldString("dsn")
	if err != nil {
		return nil, err
	}

	queryStatic, err := conf.FieldString("query")
	if err != nil {
		return nil, err
	}

	var argsMapping *bloblang.Executor
	if conf.Contains("args_mapping") {
		if argsMapping, err = conf.FieldBloblang("args_mapping"); err != nil {
			return nil, err
		}
	}

	connSettings, err := connSettingsFromParsed(conf)
	if err != nil {
		return nil, err
	}
	return newSQLRawOutput(logger, driverStr, dsnStr, queryStatic, argsMapping, connSettings), nil
}

func newSQLRawOutput(
	logger *service.Logger,
	driverStr, dsnStr string,
	queryStatic string,
	argsMapping *bloblang.Executor,
	connSettings connSettings,
) *sqlRawOutput {
	return &sqlRawOutput{
		logger:       logger,
		shutSig:      shutdown.NewSignaller(),
		driver:       driverStr,
		dsn:          dsnStr,
		queryStatic:  queryStatic,
		argsMapping:  argsMapping,
		connSettings: connSettings,
	}
}

func (s *sqlRawOutput) Connect(ctx context.Context) error {
	s.dbMut.Lock()
	defer s.dbMut.Unlock()

	if s.db != nil {
		return nil
	}

	var err error
	if s.db, err = sqlOpenWithReworks(s.logger, s.driver, s.dsn); err != nil {
		return err
	}

	s.connSettings.apply(s.db)

	go func() {
		<-s.shutSig.CloseNowChan()

		s.dbMut.Lock()
		_ = s.db.Close()
		s.dbMut.Unlock()

		s.shutSig.ShutdownComplete()
	}()
	return nil
}

func (s *sqlRawOutput) WriteBatch(ctx context.Context, batch service.MessageBatch) error {
	s.dbMut.RLock()
	defer s.dbMut.RUnlock()

	for i := range batch {
		var args []interface{}
		resMsg, err := batch.BloblangQuery(i, s.argsMapping)
		if err != nil {
			return err
		}

		iargs, err := resMsg.AsStructured()
		if err != nil {
			return err
		}

		var ok bool
		if args, ok = iargs.([]interface{}); !ok {
			return fmt.Errorf("mapping returned non-array result: %T", iargs)
		}

		if _, err = s.db.ExecContext(ctx, s.queryStatic, args...); err != nil {
			return err
		}
	}
	return nil
}

func (s *sqlRawOutput) Close(ctx context.Context) error {
	s.shutSig.CloseNow()
	s.dbMut.RLock()
	isNil := s.db == nil
	s.dbMut.RUnlock()
	if isNil {
		return nil
	}
	select {
	case <-s.shutSig.HasClosedChan():
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}