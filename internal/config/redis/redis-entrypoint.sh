#!/bin/sh

redis-server --save "" --appendonly no &

echo "🔄 Wait..."
until redis-cli ping | grep PONG; do
  sleep 1
done

echo "✅ Redis is ready!"

echo "🔄 Creating a stream 'releasy_jobs' e group 'releasy-group'..."
redis-cli XGROUP CREATE releasy_jobs releasy-group \$ MKSTREAM || echo "ℹ️ Group already exists, ignoring."

echo "🚀 it's ready!"
wait
