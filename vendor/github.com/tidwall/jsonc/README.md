# jsonc

[![GoDoc](https://img.shields.io/badge/api-reference-blue.svg?style=flat-square)](https://pkg.go.dev/github.com/tidwall/jsonc) 

jsonc is a Go package that converts the jsonc format to standard json.

The jsonc format is like standard json but allows for comments and trailing
commas, such as:

```js
{

  /* Dev Machine */
  "dbInfo": {
    "host": "localhost",
    "port": 5432,          
    "username": "josh",
    "password": "pass123", // please use a hashed password
  },

  /* Only SMTP Allowed */
  "emailInfo": {
    "email": "josh@example.com", // use full email address
    "password": "pass123",
    "smtp": "smpt.example.com",
  }

}
```

There's a provided function `jsonc.ToJSON`, which does the conversion.

The resulting JSON will always be the same length as the input and it will
include all of the same line breaks at matching offsets. This is to ensure
the result can be later processed by a external parser and that that
parser will report messages or errors with the correct offsets.

## Getting Started

### Installing

To start using jsonc, install Go and run `go get`:

```sh
$ go get -u github.com/tidwall/jsonc
```

This will retrieve the library.

### Example

The following example uses a JSON document that has comments and trailing
commas and converts it just prior to unmarshalling with the standard Go
JSON library.

```go

data := `
{
  /* Dev Machine */
  "dbInfo": {
    "host": "localhost",
    "port": 5432,          // use full email address
    "username": "josh",
    "password": "pass123", // use a hashed password
  },
  /* Only SMTP Allowed */
  "emailInfo": {
    "email": "josh@example.com",
    "password": "pass123",
    "smtp": "smpt.example.com",
  }
}
`

err := json.Unmarshal(jsonc.ToJSON(data), &config)

```

### Performance

It's fast and can convert GB/s of jsonc to json.

## Contact

Josh Baker [@tidwall](http://twitter.com/tidwall)

## License

jsonc source code is available under the MIT [License](/LICENSE).
