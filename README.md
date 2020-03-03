# Example of difficulty testing plugins

This project contains an example of a potential strange interaction between `go
test` and plugins.

Reproduction:


```shell
go test codeloader/codeloader*.go
```

The plugin is able to affect a package imported by the test package, but
multiple copies of the package under test appear to be instantiated.
