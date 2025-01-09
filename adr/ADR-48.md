# JetStream Distributed Counter CRDT

| Metadata | Value      |
|----------|------------|
| Date     | 2025-01-09 |
| Author   | @ripienaar |
| Status   | Approved   |
| Tags     | jetstream  |

| Revision | Date       | Author     | Info                    |
|----------|------------|------------|-------------------------|
| 1        | 2025-01-09 | @ripienaar | Document Initial Design |

## Context and Motivation

We wish to provide a distributed counter that will function in Clusters, Super Clusters, through Sources and any other way that data might reach a Stream.

We will start with basic addition and subtraction primitives which will automatically behave as a CRDT and be order independent.

## Solution Overview

A Stream can opt-in to supporting Counters which will allow any key to be a counter.

Publishing a message to such a Stream will load the most recent message on the subject, increment its value and save 
the new message with the latest value in the body. The header is preserved for downstream processing by Sources and for 
visibility and debugging purposes.

```bash
$ nats s get COUNTER --last-for counter.hits
Item: COUNTER#22802062 received 2025-01-09 18:05:07.93747413 +0000 UTC on Subject counter.hits

Headers:
  Nats-Incr: +2

{"val":"100"}
$ nats pub counter.hits '' -J -H "Nats-Incr:+1"
$ nats s get COUNTER --last-for counter.hits
Item: COUNTER#22802063 received 2025-01-09 18:06:00 +0000 UTC on Subject counter.hits

Headers:
  Nats-Incr: +1

{"val":"101"}
```

## Design and Behavior

The goal is to support addition and subtraction only and that a Stream that Sources 10s of other Streams all with 
counters will effectively create a big combined counter holding totals contributed to by all sources.

Handling published messages has the follow behavior and constraints:

 * A message with the header set and a non nil body is rejected
 * The header holds values like `+1`, `-1` and `-10`, in other words any valid `int64`, if the value fails to parse 
   the message is rejected with an error
 * When publishing a message to the subject the last value is loaded, the body is parsed, incremented and written 
   into the new message body. The headers are all preserved.
 * If the addition will overflow a `int64` in either direction the message will be rejected with an error
 * When publishing a message and the previous message do not have a `Nats-Incr` header it means the key is not a 
   counter, an error is returned to the user and the message is rejected
 * When a message with the header is received over a Source, that has the configuration setting enabled, processing is 
   done as above otherwise the message is discarded
 * When a message with the header is published to a Stream without the option set the message is rejected with an error
 * When a message with the header is received over a Mirror the message is stored verbatim
 * When a message without the header is received and the previous message is a counter this will be rejected since 
   the subject is then treated as being a counter only. This could be very expensive on Streams with many 
   non-counter messages being written to them

The value in the body is stored in a struct with the following keys:

```go
type CounterValue struct {
	Value string `json:"val"`
}
```

## Stream Configuration

Weather or not a Stream support this behavior should be a configuration opt-in. We want clients to definitely know
when this is supported which the opt-in approach with a boolean on the configuration would make clear.

```golang
type StreamConfig struct {
	// AllowMsgCounter enables the feature
	AllowMsgCounter bool          `json:"allow_msg_counter"`
}
```

Setting this on a Mirror should cause an error.

This feature can be turned off and on using Stream edits.

A Stream with this feature on should require API level 1.
