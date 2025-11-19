# API 및 데이터 명세

## Protocol Buffers (Protobuf)
내부 통신(Kafka 및 gRPC)에는 타입 안전성과 성능을 위해 Protobuf를 사용합니다.

### `market_data.proto`
```protobuf
syntax = "proto3";

package market_data;

message TickData {
  string symbol = 1;
  double price = 2;
  int64 volume = 3;
  int64 timestamp = 4; // Unix nanoseconds
}

message OrderBook {
  string symbol = 1;
  repeated Level bids = 2;
  repeated Level asks = 3;
  int64 timestamp = 4;
}

message Level {
  double price = 1;
  int64 quantity = 2;
}
```

### `order.proto`
```protobuf
syntax = "proto3";

package order;

enum Side {
  BUY = 0;
  SELL = 1;
}

enum OrderType {
  LIMIT = 0;
  MARKET = 1;
}

message OrderRequest {
  string id = 1;
  string symbol = 2;
  Side side = 3;
  OrderType type = 4;
  double price = 5;
  int64 quantity = 6;
  string strategy_id = 7;
}

message OrderResponse {
  string order_id = 1;
  string status = 2; // FILLED, REJECTED, PARTIAL
  double filled_price = 3;
  int64 filled_quantity = 4;
  string message = 5;
}
```

## Kafka 토픽
-   `market.tick`: `symbol`로 파티셔닝. Key=`symbol`, Value=`TickData` (Proto).
-   `market.orderbook`: `symbol`로 파티셔닝. Key=`symbol`, Value=`OrderBook` (Proto).
-   `strategy.signals`: 인지/전략 에이전트가 생성.
-   `trade.orders`: (선택사항) 전송된 모든 주문 로그.

## gRPC 서비스
### `OrderService`
```protobuf
service OrderService {
  // 저지연을 위한 양방향 스트리밍
  rpc StreamOrders(stream OrderRequest) returns (stream OrderResponse);
}
```
