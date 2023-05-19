# Ensign-Sonar

A quick QA utility to send and receive events using Ensign.


## Running Ensonar

Make sure you have your `.env` file setup using the `.env.template` file.

In one terminal:

```
$ go run ./cmd/ensonar/ sonar 2>/dev/null
```

Note: this redirects `stderr` to `/dev/null` so that errors aren't printed; you can also direct to a file to find out what is going wrong with the publisher.

In a second terminal:

```
$ go run ./cmd/ensonar listen
```
