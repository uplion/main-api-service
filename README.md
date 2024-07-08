# MainAPIService

This service accepts client API requests, processes them, and sends the data to a message queue. The worker node processes the data and sends the response back to the client.

## Environment Variables

1. `PORT`: The port on which the service listens, default is 8080.
2. `PULSAR_URL`: The URL to connect to Pulsar.
3. `PULSAR_TOKEN`: The token to connect to Pulsar.
4. `TIMEOUT`: The client timeout duration, default is 5 minutes. The format should follow the specification at https://pkg.go.dev/time#ParseDuration.
5. `DEBUG`: When set, the service runs in debug mode.

## Run

First, you should start Pulsar, and then run the service in host mode:

```bash
docker run -it \
  -e PULSAR_URL=pulsar://localhost:6650 \
  --network host \
  youxam/uplion-main:latest
```

The service listens on port 8080 by default.

Alternatively, you can run it in bridge mode:

```bash
# Map the container's port 8080 to the host's port 8080
# Replace <HOST_IP> with the actual host IP address
docker run -it \
  -p 8080:8080 \
  -e PULSAR_URL=pulsar://<HOST_IP>:6650 \
  youxam/uplion-main:latest
```

You can read the [worker](./test/worker/README.md) documentation to learn how to run the test worker node. If the worker node starts successfully, you can now call the API using the following commands:

```bash
bash test/ai.sh
```

or

```bash
bash test/ai.sh stream
```

This will invoke the API with the respective parameters.

## API Specification

### Message Queue Data

Data is sent to the topic `"model-${modelName}"`, for example, `model-gpt-3.5-turbo`.

The data is in JSON format and has the following structure:

```json5
{
  "request_id": string,
  "stream": boolean,
  "endpoint": string,
  "data": {
    // The body of the original request remains unchanged
    // Refer to https://platform.openai.com/docs/api-reference/chat for format details
  }
}
```

#### Example

##### stream=true

```json
{
  "request_id": "4e4a670a-b66a-481f-a37a-9b77af6c047e",
  "stream": true,
  "endpoint": "http://api-01.api.default.svc.cluster.local/res/ws",
  "data": {
    "model": "gpt-4",
    "messages": [
      {
        "role": "system",
        "content": "You are a helpful assistant."
      },
      {
        "role": "user",
        "content": "Hello!"
      }
    ],
    "stream": true
  }
}
```

##### stream=false

```json
{
  "request_id": "4e4a670a-b66a-481f-a37a-9b77af6c047e",
  "stream": false,
  "endpoint": "http://api-01.api.default.svc.cluster.local/res",
  "data": {
    "model": "gpt-4",
    "messages": [
      {
        "role": "system",
        "content": "You are a helpful assistant."
      },
      {
        "role": "user",
        "content": "Hello!"
      }
    ]
  }
}
```

The values of `.stream` and `.data.stream` are generally the same (if `.data.stream` is not set, `.stream` is `false`). When `true`, the worker node must send data in real-time through WebSocket. Even if the upstream API/local model does not support streaming, the worker node should still send responses in WebSocket and streaming format. The timing for establishing the WebSocket can be flexible (e.g., if streaming is not supported but requested, the WebSocket can be established after the response is complete to send the result).

### Response Format

#### stream=true

**Request Not Completed**

```json5
{
  "request_id": string,
  "end": false,
  "data": {
    // The body of the original response fragment remains unchanged
  }
}
```

**Request Completed**

```json5
{
  "request_id": string,
  "end": true,
  "data": {
    // The body of the original response fragment remains unchanged
  }
}
```

#### stream=false

```json5
{
  "request_id": string,
  "data": {
    // The body of the original response remains unchanged
  }
}
```


## Health Check

The service provides a health check endpoint at `/health`. 

```bash
❯ curl http://localhost:8080/health
{"status":"ok"}
```