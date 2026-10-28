# 渠道额外设置说明

该配置用于设置一些额外的渠道参数，可以通过 JSON 对象进行配置。主要包含以下三个设置项：

1. force_format
    - 用于标识是否对数据进行强制格式化为 OpenAI 格式
    - 类型为布尔值，设置为 true 时启用强制格式化

2. proxy
    - 用于配置网络代理
    - 类型为字符串，支持 `http`、`https`、`socks5` 和 `socks5h` 协议
    - 保存时必须包含协议和主机；仅允许空路径或根路径 `/`，不允许 query 或 fragment
    - SOCKS 代理未填写端口时，运行时使用默认端口 `1080`

3. thinking_to_content
   - 用于标识是否将思考内容`reasoning_content`转换为`<think>`标签拼接到内容中返回
   - 类型为布尔值，设置为 true 时启用思考内容转换

4. client_disconnect_drain_enabled
   - 客户端断开后继续排水：流式请求在客户端断开后不立刻关闭上游，继续读到上游自己结束，再按上游返回的真实 usage 计费
   - 类型为布尔值，默认 false（客户端断开即关闭上游连接，上游停止生成）
   - 对所有中继路径生效（Chat Completions、Claude Messages、Gemini、Responses、图片、音频等），包括开启透传的渠道；按哪种协议解析上游由渠道的上游 API 类型对应的适配器决定
   - 排水在以下任一情况结束：上游正常结束或报错；上游空闲超过 `STREAMING_TIMEOUT`；排水时长超过 `STREAM_DRAIN_MAX_SECONDS`（默认 1800 秒）
   - 透传渠道的 OpenAI Chat 流式请求在开启本项时（且渠道支持 `stream_options`、`FORCE_STREAM_OPTION` 未关闭），会在转发的请求体里补上 `stream_options.include_usage=true`，让上游返回 usage；客户端未要求 usage 时不会看到这一块
   - 客户端断开后不再切换到其它渠道重试

--------------------------------------------------------------

## JSON 格式示例

以下是一个示例配置，启用强制格式化并设置了代理地址：

```json
{
    "force_format": true,
    "thinking_to_content": true,
    "proxy": "socks5://proxy.example:1080"
}
```

--------------------------------------------------------------

通过调整上述 JSON 配置中的值，可以灵活控制渠道的额外行为，比如是否进行格式化以及使用特定的网络代理。

## 升级兼容性

`client_disconnect_drain_enabled` 取代了旧键 `responses_client_disconnect_drain_enabled`（旧键只作用于 `/v1/responses`）。运行时两者任一为 true 即开启，并且对所有路径生效；数据库不会被改写，在管理页再次保存渠道时会写入新键并去掉旧键。

旧版本会忽略代理地址中的 path、query 和 fragment。为避免升级后中断已有渠道流量，运行时会继续剥离这些遗留后缀，并对同一代理地址每个进程记录一次不含凭证和后缀的警告。该兼容逻辑不会改写数据库；再次保存渠道时必须按上述严格规则修正代理地址。

代理连接使用 30 秒 TCP 拨号超时和 30 秒 KeepAlive；TLS 握手超时为 10 秒。这些超时同样适用于未配置渠道代理的中转请求。
