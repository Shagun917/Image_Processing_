# Store Image Processing Application

## Description

This application processes store visits and images, calculating the perimeter of each image. It provides an API to submit jobs and check their status. The application is written in Go and uses Go Modules for dependency management. It can be run both locally and inside a Docker container.

## Assumptions

- The application calculates the actual image height and width instead of using random values.
- It uses a simple in-memory store master for demonstration purposes.
- The application assumes that the store IDs provided in the visits exist in the store master.

## Installation and Testing Instructions

### Prerequisites

- **Go**: Version 1.24.1
- **Docker**: Version 27.5.1
- **VS Code**: Recommended IDE

### Without Docker

1. Ensure you have Go installed on your system.
2. Clone the repository:
   ```sh
   git clone <repository-url>
   cd <repository-folder>
   ```
3. Run the application:
   ```sh
   go run main.go
   ```
4. The application will be available at [http://localhost:8080](http://localhost:8080).

### With Docker

1. Ensure you have Docker installed on your system.
2. Clone the repository:
   ```sh
   git clone <repository-url>
   cd <repository-folder>
   ```
3. Navigate to the project directory.
4. Build the Docker image:
   ```sh
   docker build -t my-app .
   ```
5. Run the Docker container:
   ```sh
   docker run -p 8080:8080 my-app
   ```
6. The application will be available at [http://localhost:8080](http://localhost:8080).

## Testing

You can test the application using **curl** or any API testing tool like **Postman**.

### Submit a Job

```sh
curl -X POST http://localhost:8080/submit/ -d '{
  "count": 1,
  "visits": [
    {
      "store_id": "S00339218",
      "image_url": ["https://example.com/image.jpg"],
      "visit_time": "2023-10-01T12:00:00Z"
    }
  ]
}' -H "Content-Type: application/json"
```

### Check the Job Status

```sh
curl http://localhost:8080/status?jobid=1
```

## Work Environment

- **Operating System**: macOS
- **Text Editor/IDE**: Visual Studio Code
- **Libraries Used**:
  - Standard Go libraries
  - `net/http`: For handling HTTP requests and responses
  - `encoding/json`: For encoding and decoding JSON data
  - `image`, `image/jpeg`, `image/png`, `image/gif`: For processing images
  - `sync`: For synchronization primitives like mutexes and wait groups
  - `math/rand`: For generating random numbers
  - `time`: For handling time-related operations
  - `log`: For logging
  - `os`: For file and directory operations

## Future Improvements

- **Additional States**: Implement a queue and add a "Queued" state/response if the job is not "Ongoing" due to system overload.
- **Additional Image Format Support**: Add support for more image formats.
- **Storage**: Implement persistent storage for jobs and store master using a database like MongoDB.
- **Authentication and Authorization**: Add authentication and authorization mechanisms to secure the API.
