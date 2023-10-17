## trie

A minimal implementation of a [trie data structure][trie] for [Go]. Differs
from most implementations in that it uses string slices (`[]string`) as keys,
rather than just strings.

This makes it suitable for efficiently storing information about hierarchical
systems in general, rather than being specifically geared towards string lookup.

Read the documentation on [godoc.org][docs] for details of how to use `trie`.

[docs]: http://godoc.org/github.com/alphagov/router/trie
[go]: http://golang.org
[trie]: https://en.wikipedia.org/wiki/Trie
