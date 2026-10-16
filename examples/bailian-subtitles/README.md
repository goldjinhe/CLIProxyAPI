# Bailian subtitle gateway

This example exposes two stable, application-facing model aliases through
CLIProxyAPI:

- `drama-subtitle-semantic` for Chinese ASR correction and disambiguation.
- `drama-subtitle-translation` for JSON-preserving multilingual translation.

Both aliases intentionally point to the pinned `qwen-plus-2025-12-01` snapshot.
The proxy forces `enable_thinking=false` because Bailian JSON mode is not
reliable in thinking mode. The application can change providers or model
snapshots later without changing its own model names.

## Configure

1. Copy `config.example.yaml` to an ignored `config.yaml`.
2. Replace both placeholder keys. Use one random gateway client key for the
   drama worker and the DashScope key for the upstream credential.
3. Start CLIProxyAPI with Docker Compose.
4. Set the drama worker endpoint to
   `http://cli-proxy-api:8317/v1/chat/completions` while both services share the
   private `orange-ai-gateway` Docker network.

Do not expose port 8317 to the public internet for application-to-gateway
traffic. The shared Docker network is internal; CLIProxyAPI keeps its default
network as a separate egress path to Bailian.

This gateway handles text correction and translation only. Subtitle ASR needs
word or sentence timestamps and must remain behind the drama platform's ASR
adapter.

See Alibaba Cloud's documentation for [OpenAI-compatible Chat
Completions](https://help.aliyun.com/zh/model-studio/qwen-api-via-openai-chat-completions)
and [structured output](https://help.aliyun.com/zh/model-studio/qwen-structured-output).
