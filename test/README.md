# Test

This directory contains related programs for testing purposes.

## 1. API Testing

Use the following commands to initiate a simple request to the service:

```bash
bash test/api.sh
```

For testing stream transmission, use the following command:

```bash
bash test/api.sh stream
```

## 2. Worker

The `test/worker` directory contains a simple program that listens to a message queue, processes messages, and sends the results back to the service. For more details, please refer to the [worker](./worker/README.md).
