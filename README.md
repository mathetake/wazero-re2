This demonstrates how to use [re2](https://github.com/google/re2), a regular expression library written in C++, through FFI based on WebAssembly with [wazero](https://github.com/tetratelabs/wazero).

Note: this is intended to be used as a demonstration for [my talk at GopherCon 2022](https://www.gophercon.com/agenda/session/944206).


Here's a benchmark result in [lib_test.go](./lib_test.go) where we compare the standard `regexp` package with re2 via FFI with wazero:

```
$ benchstat stdlib.txt re2.txt 
name                                 old time/op    new time/op       delta
RegexpMatch/Hard/not_match/16B-10      3.39ns ± 1%     506.54ns ± 6%   +14839.54%  (p=0.008 n=5+5)
RegexpMatch/Hard/not_match/1KB-10      20.9µs ± 1%        0.6µs ± 3%      -97.25%  (p=0.008 n=5+5)
RegexpMatch/Hard/not_match/1MB-10      26.2ms ± 2%        0.0ms ±21%      -99.93%  (p=0.008 n=5+5)
RegexpMatch/Hard/not_match/32MB-10      835ms ± 1%          1ms ± 1%      -99.89%  (p=0.008 n=5+5)
RegexpMatch/Hard/match/16B-10           443ns ± 1%        525ns ± 1%      +18.55%  (p=0.008 n=5+5)
RegexpMatch/Hard/match/1KB-10          9.54µs ± 1%       4.05µs ± 0%      -57.56%  (p=0.008 n=5+5)
RegexpMatch/Hard/match/1MB-10          60.3ms ± 1%        3.6ms ± 1%      -93.98%  (p=0.008 n=5+5)
RegexpMatch/Hard/match/32MB-10          1.92s ± 0%        0.12s ± 0%      -93.94%  (p=0.008 n=5+5)
RegexpMatch/Hard1/not_match/16B-10     1.80µs ± 3%       0.52µs ± 1%      -71.09%  (p=0.008 n=5+5)
RegexpMatch/Hard1/not_match/1KB-10      107µs ± 1%          4µs ± 0%      -96.22%  (p=0.008 n=5+5)
RegexpMatch/Hard1/not_match/1MB-10      129ms ± 1%          4ms ± 1%      -97.17%  (p=0.008 n=5+5)
RegexpMatch/Hard1/not_match/32MB-10     4.10s ± 1%        0.12s ± 0%      -97.16%  (p=0.008 n=5+5)
RegexpMatch/Hard1/match/16B-10          304ns ± 2%        528ns ± 1%      +73.71%  (p=0.008 n=5+5)
RegexpMatch/Hard1/match/1KB-10         9.18µs ± 5%       4.06µs ± 1%      -55.78%  (p=0.008 n=5+5)
RegexpMatch/Hard1/match/1MB-10          167ms ± 2%          4ms ± 2%      -97.83%  (p=0.008 n=5+5)
RegexpMatch/Hard1/match/32MB-10         5.32s ± 1%        0.12s ± 1%      -97.81%  (p=0.008 n=5+5)
```

The benchmark cases are borrowed from [the Go official test cases](https://github.com/golang/go/blob/54182ff54a687272dd7632c3a963e036ce03cb7c/src/regexp/exec_test.go), 
and this indicates that the larger the input text is, the better wazero+re2 FFI performs. The reason why re2 FFI under-performs
with the small text is that: it has to copy the text into the Wasm's linear memory region so that the re2 can reference there
message in Wasm, which comes with the allocation each time. Therefore, the allocation can be dominant in such a case,
and makes huge difference there.

Note: test cases are chosen somewhat artificially. This doesn't necessary recommend and state that wazero+re2 performs better than the standard Go library. 
This is just a demo on the potential of FFI with Wasm!
