---
title: xml_documents
slug: xml_documents
type: scanner
status: beta
---

<!--
     THIS FILE IS AUTOGENERATED!

     To make changes please edit the corresponding source file under internal/impl/<provider>.
-->

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

:::caution BETA
This component is mostly stable but breaking changes could still be made outside of major version releases if a fundamental problem with the component is found.
:::
Consumes a stream of one or more XML documents and performs a mutation on the data.

```yml
# Config fields, showing default values
xml_documents:
  operator: ""
  cast: false
```

## Fields

### `operator`

An XML [operation](#operators) to apply to messages.


Type: `string`  
Default: `""`  
Options: `to_json`.

### `cast`

Whether to try to cast values that are numbers and booleans to the right type. Default: all values are strings.


Type: `bool`  
Default: `false`  


