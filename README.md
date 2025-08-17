# 💻 My Redis Clone

A simple, fast, and reliable in-memory key-value store designed for student projects.  
This Redis-compatible clone provides a robust solution for caching and session management without the complexity of an enterprise-grade database.

---

## ✨ Features

- **Core Redis Commands:** Supports essential commands like `GET`, `SET`, `DEL`, and `EXPIRE`.  
- **In-Memory Storage:** Blazing fast data access with all data stored in RAM.  
- **Thread-Safe:** Efficient sharding and locking mechanism for safe concurrent access.  
- **Lightweight & Easy to Use:** Minimal dependencies and simple setup.

---

## 🚀 Getting Started

Follow these instructions to run the project locally.

### Prerequisites

- Node.js (v14 or higher)  
- npm

### Installation

```bash
git clone https://github.com/your-username/your-repo-name.git
cd your-repo-name
npm install
npm start
The server will start and listen for client connections on localhost:6379.

🔧 Client Usage Examples
You can use this Redis clone in your applications with a Redis client library.

Node.js
Install the Redis client:

bash
Copy
Edit
npm install redis
Example usage:

javascript
Copy
Edit
import { createClient } from 'redis';

async function connectToRedis() {
  const client = createClient();

  client.on('error', (err) => console.log('Redis Client Error', err));

  await client.connect();
  console.log('Successfully connected to the Redis clone!');

  await client.set('my_app:user_123', '{"name": "Alice", "age": 25}');
  console.log('Set a new user in the cache.');

  const user = await client.get('my_app:user_123');
  console.log('Retrieved user:', user);

  await client.quit();
}

connectToRedis();
Python
Install the Redis client:

bash
Copy
Edit
pip install redis
Example usage:

python
Copy
Edit
import redis

def connect_to_redis():
    r = redis.Redis(host='localhost', port=6379, db=0)

    try:
        r.ping()
        print("Successfully connected to the Redis clone!")

        r.set('my_app:user_123', '{"name": "Bob", "age": 30}')
        print("Set a new user in the cache.")

        user = r.get('my_app:user_123')
        print("Retrieved user:", user.decode('utf-8'))

    except redis.exceptions.ConnectionError as e:
        print(f"Failed to connect to Redis: {e}")

connect_to_redis()
Java (Maven)
Add Jedis to your pom.xml:

xml
Copy
Edit
<dependencies>
    <dependency>
        <groupId>redis.clients</groupId>
        <artifactId>jedis</artifactId>
        <version>4.3.1</version>
    </dependency>
</dependencies>
Example usage:

java
Copy
Edit
import redis.clients.jedis.Jedis;

public class RedisCloneClient {
    public static void main(String[] args) {
        Jedis jedis = new Jedis("localhost", 6379);

        try {
            jedis.ping();
            System.out.println("Successfully connected to the Redis clone!");

            jedis.set("my_app:user_456", "{\"name\": \"Charlie\", \"age\": 35}");
            System.out.println("Set a new user in the cache.");

            String user = jedis.get("my_app:user_456");
            System.out.println("Retrieved user: " + user);

        } catch (Exception e) {
            System.err.println("Failed to connect to Redis: " + e.getMessage());
        } finally {
            if (jedis != null) jedis.close();
        }
    }
}
⚙️ Configuration
The server runs on port 6379 by default.
To change the port, set the PORT environment variable:

bash
Copy
Edit
PORT=7000 npm start
🤝 Contributing
This project is a great way to learn about databases and concurrency.
Feel free to open issues or submit pull requests with new features or bug fixes.