import { Client, type ConsumerConfig } from 'pulsar-client';
import axios from 'axios';
import WebSocket from 'ws';
import dotenv from 'dotenv';
import {createParser, type ParsedEvent, type ReconnectInterval} from 'eventsource-parser'

dotenv.config();

const topic = process.env.TOPIC
const openaiApiKey = process.env.OPENAI_API_KEY

if (!topic || !openaiApiKey) {
    console.error('Please provide a topic name and OpenAI API key.');
    process.exit(1);
}


async function handleMessage(content: string) {
    const body = JSON.parse(content);
    const { stream, endpoint, data, request_id } = body;

    console.log('Received message:');
    console.log(JSON.stringify(body, null, 2));

    if (stream) {
        const ws = new WebSocket(endpoint);

        ws.on('open', () => {
            console.log('WebSocket connection opened.');
        });

        ws.on('close', () => {
            console.log('WebSocket connection closed.');
        });

        try {
            const response = await fetch('https://api.openai.com/v1/chat/completions', {
                method: 'POST',
                body: JSON.stringify(data),
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${openaiApiKey}`
                }
            });

            const reader = response.body!.getReader();
            const parser = createParser((event: ParsedEvent | ReconnectInterval) => {
                if (event.type === 'event') {
                    console.log(event.data)
                    if (event.data === '[DONE]') {
                        ws.send(JSON.stringify({ request_id, end: true }));
                    } else {
                        ws.send(JSON.stringify({ request_id, data: JSON.parse(event.data), end: false }));
                    }
                }
            })
            
            while (true) {
                const { value, done } = await reader.read();
                if (done || !value) break;
                parser.feed(new TextDecoder().decode(value))
            }
        } catch (error) {
            console.error('Error from OpenAI:', error);
        }

    } else {
        try {
            const response = await axios.post('https://api.openai.com/v1/chat/completions', data, {
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${openaiApiKey}`
                },
            });

            try {
                await axios.post(endpoint, {
                    request_id,
                    data: response.data
                });
                console.log('Response sent to endpoint:', endpoint);
            } catch (error) {
                console.error('Error sending response:', error);
            }
        } catch (error) {
            console.error('Error from OpenAI:', (error as any).response.data);
        }
    }
}

const client = new Client({
    serviceUrl: process.env.PULSAR_URL || 'pulsar://localhost:6650',
});


; (async () => {
    const consumer = await client.subscribe({
        topic: topic!,
        subscription: 'bun-subscription'
    });

    console.log('Subscribed to topic:', topic);

    while (true) {
        const msg = await consumer.receive();
        const content = msg.getData().toString('utf-8');
        await handleMessage(content);
        consumer.acknowledge(msg);
    }
})().catch(console.error);