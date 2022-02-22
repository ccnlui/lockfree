## Summary

### `mpmc.go`
This is an implementation of Dimitry's MPMC queue (original design [here](https://www.1024cores.net/home/lock-free-algorithms/queues/bounded-mpmc-queue)). This file is a copy-and-paste from a [blog](https://bravenewgeek.com/so-you-wanna-go-fast/), whose author translated the original c++ design to [golang](https://github.com/Workiva/go-datastructures/blob/master/queue/ring.go). He added some extra non-blocking methods, `Offer()` and `Poll()`. There is a bug in `Offer()`: it is not guaranteed that the queue is full when `Offer()` returns false, not when there are multiple producers. Although it is true in the case of a single producer.

### `dspsc.go`
Dimitry's MPMC queue turned into a SPSC queue. Seems to be the fastest.

### `spsc.go`
Naive attempt at a SPSC queue. This is still faster than a channel by about 3 times.

### `bspsc.go`
Incorrect attempt to optimize `spsc.go` by batching. Perhaps can be fixed by flushing read/write at a fixed interval during low traffic.

### `cspsc.go`
Attempt to optimize `spsc.go` by caching read/write index. Seems to faster than original by about 2 times.
