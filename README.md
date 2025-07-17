# Riddle worker for stränge.de riddle generation

This repository contains the **riddle worker** microservice for generating riddles for [strangui](https://github.com/polarfoxDev/strangui).

## Overview

The riddle worker is a Go-based service that automates the creation of riddles from riddle concepts. It interacts with Redis to query for jobs and processes them by trying to generate a valid riddle from the concept. The results are then pushed to the Redis queue for further processing by other services.

## Features

- **Automated riddle generation** from pre-generated concepts
- **Queue management** via Redis
- **Logging** with logrus for easy debugging and monitoring

## How It Works

1. The worker monitors the Redis queue `generate-riddle`.
2. Take a job from the queue, which contains a riddle concept.
3. Process the job by generating a riddle from the concept.
4. Push the generated riddle back to the Redis queue for further processing.

## Project Structure

- [`main.go`](./main.go): Main worker loop, Redis integration, configuration, and logging.
- [`worker.go`](./worker.go): Contains the logic for processing riddle generation jobs, allows for parallel execution.
- [`m/convert/format.go`](./convert/format.go): Transformation utility to convert to the output format that `strangui` requires.
- [`m/defaults/colors.go`](./defaults/colors.go): Contains default color definitions for debugging formatting.
- [`m/models`](./models): Defines models used in the application.
- [`m/models/riddle.go`](./models/riddle.go): Defines the Riddle model used in the application. This includes most of the actual logic for generating riddles from concepts.
- [`m/random/random.go`](./random/random.go): Contains a utility function to prepare a secure random number generator.

## Setup

### Prerequisites

- Go 1.20+
- Redis server

### Installation

Clone the repository and install dependencies:

```bash
git clone https://github.com/polarfoxDev/straenge-riddle-worker.git
cd straenge-riddle-worker
go mod tidy
```

### Configuration

Create a `.env` file in the root directory with the following variables:

```env
REDIS_URL=localhost:6379
LOG_LEVEL=info
PARALLEL_COUNT=1
JOB_TIMEOUT_SECONDS=600
RETRY_TIMEOUT_SECONDS=600
```

### Running the Worker

Start the worker:

```bash
go run main.go
```

## Contributing

Any contributions you make are greatly appreciated.

If you have a suggestion that would make this better, please fork the repo and create a pull request. You can also simply open an issue.
Don't forget to give the project a star! Thanks!

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

Distributed under the **MIT** License. See [`LICENSE`](./LICENSE) for more information.
