This is a script I use to backup file servers 

## TODO 

- [ ] Add client side rate limiting
- [ ] Use a real solution for meta-info storage (consumes too much disk space as is)

## Usage
```
Usage of cp-http:
  -d int
        maximum crawl depth (default 20)
  -r string
        file server root url (default "http://localhost:8000/")
  -t int
        HTTP requests timeout in seconds (default 20)
  -w int
        number of workers (maximum concurrent HTTP requests) (default 10)
```
