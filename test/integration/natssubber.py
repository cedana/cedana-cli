import time
import argparse
import nats


async def message_handler(msg):
    global message_count
    message_count += 1


async def subscribe_nats_stream(url, subject):
    global message_count
    message_count = 0

    # Connect to NATS server
    nc = await nats.connect()

    # Subscribe to the specified subject
    await nc.subscribe(subject, cb=message_handler)

    # Subscribe for 10 seconds
    await asyncio.sleep(10)

    # Disconnect from NATS server
    await nc.close()

    return message_count

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="NATS Stream Subscriber")
    parser.add_argument("url", help="NATS server URL")
    parser.add_argument("subject", help="Subject to subscribe to")
    parser.add_argument(
        "output_file", help="Output file to write message count")

    args = parser.parse_args()

    import asyncio
    message_count = asyncio.run(subscribe_nats_stream(args.url, args.subject))

    with open(args.output_file, "w") as f:
        f.write(str(message_count))
