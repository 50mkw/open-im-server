# OpenIM 多集群 Kafka 支持

## 背景

原始 OpenIM 只支持连接单一 Kafka 集群。本改造允许通过配置文件指定任意数量的 Kafka 集群，
消息按 `hash(conversationID) % N` 路由到对应集群，集群数量可随时增减，无需改代码。

---

## 代码改动位置

### 新增文件

| 文件 | 说明 |
|------|------|
| `pkg/common/kafka/interface.go` | 定义 `MessageProducer` 和 `ConsumerGroupHandler` 接口 |
| `pkg/common/kafka/multi_producer.go` | `MultiProducer`：`hash(key) % N` 选集群发消息 |
| `pkg/common/kafka/multi_consumer.go` | `MultiConsumerGroup`：每集群一个 goroutine，fan-in 同一 handler |

### 修改文件

| 文件 | 改动说明 |
|------|---------|
| `pkg/common/config/config.go` | `Kafka` struct 新增 `Clusters []KafkaCluster`、`BuildClusters()` 方法 |
| `pkg/common/storage/controller/msg.go` | producer 改用 `MultiProducer` |
| `pkg/common/storage/controller/msg_transfer.go` | producer 改用 `MultiProducer` |
| `pkg/common/storage/controller/push.go` | producer 改用 `MultiProducer`，`NewPushDatabase` 返回 error |
| `internal/push/push.go` | 适配 `NewPushDatabase` 新增的 error 返回值 |
| `internal/msgtransfer/online_history_msg_handler.go` | consumer 改用 `MultiConsumerGroup` |
| `internal/msgtransfer/online_msg_to_mongo_handler.go` | consumer 改用 `MultiConsumerGroup` |
| `internal/push/push_handler.go` | consumer 改用 `MultiConsumerGroup` |
| `internal/push/offlinepush_handler.go` | consumer 改用 `MultiConsumerGroup` |
| `config/kafka.yml` | 新增 `clusters` 配置块及使用示例 |

---

## 路由原理

```
Producer 发消息：
  hash(conversationID) % len(clusters) → 选第 N 个集群
  同一 conversationID 永远落在同一集群，保证消息顺序

Consumer 消费：
  每个集群各启动一个 ConsumerGroup goroutine
  所有集群的消息 fan-in 到同一个 handler 处理
```

---

## 配置方式（config/kafka.yml）

### 单集群（默认，向后兼容）

不设置 `clusters`，沿用原有 `address` 和 `tls` 字段：

```yaml
address: [ kafka-xxxxx.aivencloud.com:26594 ]
tls:
  enableTLS: true
  caCrt: /openim-server/config/kafka-certs/ca.crt
  clientCrt: /openim-server/config/kafka-certs/client.crt
  clientKey: /openim-server/config/kafka-certs/client.key
clusters: []
```

### 多集群（两个 Aiven 账号示例）

设置 `clusters` 后，`address` 和 `tls` 字段自动忽略：

```yaml
username: ''
password: ''
producerAck: ''
compressType: none
clusters:
  - address: [ kafka-account1.aivencloud.com:26594 ]
    tls:
      enableTLS: true
      caCrt: /openim-server/config/kafka-certs/account1/ca.crt
      clientCrt: /openim-server/config/kafka-certs/account1/client.crt
      clientKey: /openim-server/config/kafka-certs/account1/client.key
      insecureSkipVerify: false
  - address: [ kafka-account2.aivencloud.com:26594 ]
    tls:
      enableTLS: true
      caCrt: /openim-server/config/kafka-certs/account2/ca.crt
      clientCrt: /openim-server/config/kafka-certs/account2/client.crt
      clientKey: /openim-server/config/kafka-certs/account2/client.key
      insecureSkipVerify: false
```

### 增减集群

- **增加集群**：在 `clusters` 末尾追加一项，重新构建镜像并重启
- **删除集群**：从 `clusters` 中删除对应项，重新构建镜像并重启

> ⚠️ 增减集群会改变 `hash(conversationID) % N` 的结果，导致部分会话切换到不同集群。
> 建议在低峰期操作，且确保所有集群的 topic 都已预先创建。

---

## 构建镜像

```bash
cd ~/claude_code/open-im-server
docker build -t openim/openim-server:multicluster .
```

构建时间约 5~10 分钟（首次需下载依赖）。

---

## 部署到 openim-docker

### 首次部署 / 更新代码后重新部署

```bash
# 1. 构建新镜像
cd ~/claude_code/open-im-server
docker build -t openim/openim-server:multicluster .

# 2. 重启 openim-server（其他服务不受影响）
cd ~/openim-docker
docker compose up -d --no-deps openim-server

# 3. 查看启动日志确认正常
docker logs openim-server 2>&1 | grep -E "running normally|kafka|Error"
```

### 回滚到原始镜像

编辑 `~/openim-docker/.env`，将 `OPENIM_SERVER_IMAGE` 改回原版：

```bash
# 编辑 .env
OPENIM_SERVER_IMAGE=openim/openim-server:v3.8.3-patch.9

# 重启
cd ~/openim-docker
docker compose up -d --no-deps openim-server
```

---

## 注意事项

1. **每个集群的 topic 需提前创建**（toRedis / toMongo / toPush / toOfflinePush）
2. **Aiven 免费套餐每个 topic 最多 2 个分区**，多集群可突破单集群分区上限
3. **`username` / `password` / `producerAck` / `compressType` 为所有集群共用**，
   各集群只能单独配置 `address` 和 `tls`
4. **证书文件需通过 docker-compose volumes 挂载到容器内**，路径与 `kafka.yml` 中一致
