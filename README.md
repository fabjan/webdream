# Web Dream

A fake web site dreamt up by a large language model.

Visit any URL, and it should have a response.

Warning: don't trust any of the content.
Double check any links it gives you.

## Requirements

You need a Groq API key to run Web Dream.

## Getting Started

```sh
go build
./webdream
```

Browse to `http://localhost:3000/` to see the site.

## Configuration

| Variable     | Description                                      |
|:-------------|:-------------------------------------------------|
| PORT         | <port> number to serve HTTP form on any interface
| ADDR         | <ip>:<port> to serve HTTP from (supercedes PORT)
| GROQ_API_KEY | Your API key for Groq

## Future Work

- Support other LLM APIs
- Containerize
- CSS?

## License

This code is licensed under the Apache 2.0 license. See LICENSE for details.
