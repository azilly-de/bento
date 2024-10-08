package couchbase

import (
	"errors"

	"github.com/couchbase/gocb/v2"
)

func valueFromOp(op gocb.BulkOp) (out any, cas gocb.Cas, err error) {
	switch o := op.(type) {
	case *gocb.GetOp:
		if o.Err != nil {
			return nil, gocb.Cas(0), o.Err
		}
		err := o.Result.Content(&out)

		return out, o.Result.Cas(), err
	case *gocb.InsertOp:
		if o.Result != nil {
			return nil, o.Result.Cas(), o.Err
		}
		return nil, gocb.Cas(0), o.Err
	case *gocb.RemoveOp:
		if o.Result != nil {
			return nil, o.Result.Cas(), o.Err
		}
		return nil, gocb.Cas(0), o.Err
	case *gocb.ReplaceOp:
		if o.Result != nil {
			return nil, o.Result.Cas(), o.Err
		}
		return nil, gocb.Cas(0), o.Err
	case *gocb.UpsertOp:
		if o.Result != nil {
			return nil, o.Result.Cas(), o.Err
		}
		return nil, gocb.Cas(0), o.Err
	}

	return nil, gocb.Cas(0), errors.New("type not supported")
}

func get(key string, _ []byte, _ gocb.Cas) gocb.BulkOp {
	return &gocb.GetOp{
		ID: key,
	}
}

func insert(key string, data []byte, _ gocb.Cas) gocb.BulkOp {
	return &gocb.InsertOp{
		ID:    key,
		Value: data,
	}
}

func remove(key string, _ []byte, cas gocb.Cas) gocb.BulkOp {
	return &gocb.RemoveOp{
		ID:  key,
		Cas: cas,
	}
}

func replace(key string, data []byte, cas gocb.Cas) gocb.BulkOp {
	return &gocb.ReplaceOp{
		ID:    key,
		Value: data,
		Cas:   cas,
	}
}

func upsert(key string, data []byte, cas gocb.Cas) gocb.BulkOp {
	return &gocb.UpsertOp{
		ID:    key,
		Value: data,
		Cas:   cas,
	}
}
