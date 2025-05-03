# ADR-001: Use TimescaleDB Instead of Kafka for Ticker Price Storage

## Status

Accepted

## Context

The original architecture of the `gold-digger` application used Kafka (Redpanda) to decouple data ingestion and signal processing. This setup enabled streaming stock price data from Alpha Vantage and triggering signal evaluations via Kafka consumers.

However, the use of Kafka introduces overhead for this specific use case:
- Kafka is best suited for high-throughput, fan-out event pipelines.
- It requires operational maintenance (especially self-hosted).
- It stores data transiently unless paired with a durable sink (e.g., PostgreSQL or ClickHouse).
- The consumer group logic adds complexity for single-consumer use cases.

The `gold-digger` project polls 4–10 stock tickers from Alpha Vantage every 5 minutes and evaluates signal windows every 15 minutes. This volume and frequency are well within the capabilities of a time-series database.

## Decision

We will **remove Kafka** from the architecture and replace it with **TimescaleDB** for the following reasons:

- ✅ **Durability**: TimescaleDB persists all price data natively.
- ✅ **Queryability**: We can query, aggregate, and analyze price windows with standard SQL.
- ✅ **Simplified architecture**: No need to maintain Kafka producer/consumer code or offsets.
- ✅ **Increased reliability**: No message loss risk on consumer downtime.
- ✅ **Perfect fit for time-series data**: TimescaleDB is optimized for this exact workload.

We will:
- Remove all Kafka-related config, producer, and consumer code.
- Replace with a `TickerPriceRepository` that saves polled prices directly to TimescaleDB.
- Adjust the signal worker to either query directly from TimescaleDB or cache recent prices from DB into memory.

## Consequences

- ❌ We lose real-time pub/sub support (which is unnecessary in this case).
- ✅ We gain historical data persistence, SQL querying, and simplification.
- ✅ The system becomes easier to deploy on a self-hosted microserver with no external dependencies.

## Alternatives Considered

- Keeping Kafka and adding PostgreSQL as a sink
- Using Redis Streams for lightweight pub/sub
- Using GCP Pub/Sub or Upstash Kafka (externalized)

These were rejected due to:
- Added complexity
- No need for event replay
- The polling model being time-driven, not event-driven

---

