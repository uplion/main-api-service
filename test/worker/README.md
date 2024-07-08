# Worker

This directory contains a simple worker program that listens to a message queue, processes messages, and sends the results back to the service.

## Environment Variables

The worker program relies on the following environment variables:

- `TOPIC`: The topic of the message queue to listen to. If `ai.sh` uses `model="gpt-3.5-turbo"`, then it should listen to `TOPIC=model-gpt-3.5-turbo`.
- `OPENAI_API_KEY`: The API key for OpenAI.
- `PULSAR_URL`: The URL of the Pulsar service. The default value is `'pulsar://localhost:6650'`.

**DO NOT SUPPORT AUTHENTICATION FOR PULSAR YET**

## Running the Worker

1. Install dependencies:

   ```bash
   pnpm i
   ```

2. Start the development server:

   ```bash
   pnpm dev
   ```

3. Start the main program and message queue.

4. Run the [`ai.sh`](../ai.sh) script to see the results:

   ```bash
   bash ai.sh
   ```

This should show the results of the OpenAI API call, demonstrating that the worker program is functioning correctly.
 are correctly configured and running to verify the complete functionality of the worker program.