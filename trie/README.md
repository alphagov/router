## trie

A minimal implementation of a [trie data structure][trie] for [Go]. Differs
from most implementations in that it uses string slices (`[]string`) as keys,
rather than just strings.

This makes it suitable for efficiently storing information about hierarchical
systems in general, rather than being specifically geared towards string lookup.

See also the [Godoc documentation for
`trie`](https://pkg.go.dev/github.com/alphagov/router/trie).

[go]: https://go.dev/
[trie]: https://en.wikipedia.org/wiki/Trie
