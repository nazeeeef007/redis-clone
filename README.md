💻 My Redis Clone
A simple, fast, and reliable in-memory key-value store designed for student pet projects. This Redis-compatible clone provides a robust solution for caching and session management without the complexity of an enterprise-grade database.

✨ Features
Core Redis Commands: Supports essential commands like GET, SET, DEL, and EXPIRE.

In-Memory Storage: Blazing fast data access with all data stored in RAM.

Thread-Safe: Uses an efficient sharding and locking mechanism to ensure safe concurrent access from multiple clients.

Lightweight & Easy to Use: Minimal dependencies and a simple setup process.

🚀 Getting Started
These instructions will get you a copy of the project up and running on your local machine.

Prerequisites
To run this server, you will need:

Node.js (version 14 or higher)

npm

Installation
Clone the repository:

git clone https://github.com/your-username/your-repo-name.git
cd your-repo-name

Install dependencies:

npm install

Run the server:

npm start

The server will start and listen for client connections on localhost:6379.

🔧 Usage in Your Application
To use this Redis clone in your own CRUD application, you'll need a Redis client library for your programming language. Here is a simple example using the popular node-redis library.

1. Install the Redis client library:

npm install redis

2. Connect and use the client in your code:

import { createClient } from 'redis';

async function connectToRedis() {
  // Create a client instance
  const client = createClient();

  // Handle connection errors
  client.on('error', (err) => console.log('Redis Client Error', err));

  // Connect to the server
  await client.connect();
  console.log('Successfully connected to the Redis clone!');

  // Set a key-value pair
  await client.set('my_app:user_123', '{"name": "Alice", "age": 25}');
  console.log('Set a new user in the cache.');

  // Get the value of the key
  const user = await client.get('my_app:user_123');
  console.log('Retrieved user:', user);

  // Disconnect when you're done
  await client.quit();
}

connectToRedis();

⚙️ Configuration
The server runs on port 6379 by default. You can change this by setting the PORT environment variable before starting the server:

PORT=7000 npm start

🤝 Contributing
This project is a great way to learn about databases and concurrency. Feel free to open issues or submit pull requests with new features or bug fixes.

📄 License
This project is licensed under the MIT License.