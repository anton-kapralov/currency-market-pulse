CREATE DATABASE IF NOT EXISTS cmp;

CREATE TABLE cmp.trades
(
    user_id             String,
    currency_from       LowCardinality(String),
    currency_to         LowCardinality(String),
    amount_sell_micros  UInt64,
    amount_buy_micros   UInt64,
    rate                Float64,
    originating_country LowCardinality(String),
    time_placed         DateTime64(3, 'UTC')
)
    ENGINE = MergeTree()
        ORDER BY time_placed
