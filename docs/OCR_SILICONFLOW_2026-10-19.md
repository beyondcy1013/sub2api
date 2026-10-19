# SiliconFlow OCR 账号接入（2026-10-19）

- 范围：仅本机 openEuler sub2api 主服务18381，经管理员API新建账号325 `SiliconFlow OCR · Qwen3-VL-32B`，active/schedulable，绑定现有分组2 freex1111。free实例和远端feishuFwd配置均未修改，服务未重启。
- 上游base URL为 https://api.siliconflow.cn/v1；凭据从远端当前私有business.json读取，未写入仓库或输出。精确model_mapping仅开放 `Qwen/Qwen3-VL-32B-Instruct`，openai_capabilities仅chat_completions，Responses强制转换为Chat Completions，WebSocket关闭。
- 验证：使用现有分组2管理员用户API Key，通过本机 `/v1/chat/completions` 发送生成的测试PNG，HTTP200，正确返回 `SUB2API OCR 58427` 与 `Price 128.50`；耗时34.81秒。日志request_id df2c6fd4-724b-41ec-ad45-ed257c168681明确account_id325，账号统计累计1请求/165token。耗时包含现有request-triggered质量检测，不等于OCR上游纯推理耗时。
- 当前限制：该模型尚无网关价格表，调用统计费用记0；不代表供应商免费。未擅自填入推测价格。API Key继续使用现有sub2api分组密钥，不向客户端泄露上游密钥。
- 回滚：管理员API将账号325暂停调度/停用即可；不删除原有账号，不修改分组已有路由或其他模型。

## 经验

- 接入验证必须走真实网关并核对账号ID/模型/统计；上游直连成功不足以证明分组调度可用。
- 模型能力应与供应商协议匹配，不能把OCR图文理解当图像生成或声明原生Responses支持。网关零费用记录需与上游实际计费明确区分。
