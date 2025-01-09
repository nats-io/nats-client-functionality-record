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

# Context and Motivation

We wish to provide a distributed counter that will function in Clusters, Super Clusters, through Mirrors and Sources
and any other way that data might reach a Stream.

We will start with basic addition and subtraction primitives which will automatically behave as a CRDT and be order independent.

# Solution Overview

A Stream can opt-in to supporting Counters which will allow any key to be a counter.

Publishing a message to such a Stream will load the most recent message on the subject, increment it's value and save the new message with the latest value.

```bash
$ nats s get COUNTER --last-for counter.hits
Item: COUNTER#22802062 received 2025-01-09 18:05:07.93747413 +0000 UTC on Subject counter.hits

Headers:
  Nats-Incr: +2

{"val":100}
$ nats pub counter.hits '' -J -H "Nats-Incr:+1"
$ nats s get COUNTER --last-for counter.hits
Item: COUNTER#22802063 received 2025-01-09 18:06:00 +0000 UTC on Subject counter.hits

Headers:
  Nats-Incr: +1

{"val":101}
```

# Design and Behavior

 * When publishing a message to the subject the last value is loaded, the body is parsed, incremented and written 
   into the new message body
 * When publishing a message and the previous message do not have a `Nats-Incr` header it means the key is not a 
   counter, an error is returned to the user and the message is rejected
 * When a message with the header is received over a Source processing is done as above
 * When a message with the header is received over a Mirror the message is stored verbatim
 * When a message without the header is received and the previous message is a counter this will also be rejected, 
   though this could be very expensive on streams with many non counter messages being written to them


# Stream Configuration

Weather or not a stream support this behavior should be a configuration opt-in. We want clients to definitely know
when this is supported which the opt-in approach with a boolean on the configuration would make clear.

```golang
type StreamConfig struct {
	// AllowMsgCounter enables the feature
	AllowMsgCounter bool          `json:"allow_msg_counter"`
}
```

This feature can be turned off and on using Stream edits.

A stream with this feature on should require API level 1.
