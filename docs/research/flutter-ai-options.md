# Research: Flutter AI chat and agent options

_Checked: 2026-08-20. This report uses only official documentation, repositories, source pages, and package pages._

## Summary

Balaur should use a small, app-owned chat service with `openai_dart` and a custom `baseUrl`. This option supports Android, iOS, and Linux without an agent framework.

Keep service-owned secrets behind a backend. Store conversations in an app-owned repository. Add a bounded tool loop only when Balaur adds domain tools.

## Findings

### 1. Recommended KISS architecture

Use these layers:

1. A custom Flutter chat screen owns presentation state.
2. A small `ChatGateway` interface owns model requests and streamed events.
3. `openai_dart` implements Chat Completions against Balaur's custom endpoint.
4. An app-owned repository stores conversations and messages.
5. A backend proxy holds any service-owned API key.

`openai_dart` is a community package, not an OpenAI SDK. It supports custom base URLs, Chat Completions, Server-Sent Events (SSE), tools, structured output, and all Balaur target platforms. [Package](https://pub.dev/packages/openai_dart) [Repository](https://github.com/davidmigloz/ai_clients_dart/tree/main/packages/openai_dart)

Start with text chat and streaming only. Do not add LangChain.dart, Genkit, Model Context Protocol (MCP), or an agent runtime now.

Keep the gateway narrow enough to replace `openai_dart` with direct HTTP later. Run compatibility tests against Balaur's endpoint before release.

The endpoint must match each used Chat Completions feature. “OpenAI-compatible” does not guarantee matching streaming, tool, or structured-output behavior. [OpenAI Chat API](https://platform.openai.com/docs/api-reference/chat/create) [Function calling](https://platform.openai.com/docs/guides/function-calling)

### 2. Option comparison

| Option | Maintenance and platforms | Capabilities | Fit for Balaur |
|---|---|---|---|
| `flutter_ai_toolkit` | Official Flutter package. It lists Android, iOS, macOS, and web. Linux is not listed. | Chat UI, attachments, streaming, history serialization, and a custom `LlmProvider`. Its generic provider emits text streams. | **Reject for now.** Linux support is missing. Its narrow provider interface hides rich stream events. [Docs](https://docs.flutter.dev/ai/ai-toolkit) [Custom providers](https://docs.flutter.dev/ai/ai-toolkit/custom-llm-providers) [Package](https://pub.dev/packages/flutter_ai_toolkit) |
| `firebase_ai` | Official Firebase package in FlutterFire. It lists Android, iOS, macOS, and web. Linux is not listed. | Gemini chat, streaming, structured output, function calling, and Firebase App Check integration. | **Reject.** It does not support Linux or Balaur's OpenAI-compatible endpoint. [Package](https://pub.dev/packages/firebase_ai) [Source](https://github.com/firebase/flutterfire/tree/main/packages/firebase_ai/firebase_ai) |
| Google Gen AI Dart | The old `google_generative_ai` package is deprecated and unlisted. Its repository is archived. The current Gen AI SDK list excludes Dart. | Frozen legacy Gemini client. | **Do not use.** Google directs Flutter users to Firebase AI Logic or Genkit Dart. [Google libraries](https://ai.google.dev/gemini-api/docs/libraries) [Package](https://pub.dev/packages/google_generative_ai) [Repository](https://github.com/google/generative-ai-dart) |
| Direct HTTP plus SSE | Dart `HttpClientResponse` and `package:http` `StreamedResponse` expose streamed response bytes. Neither API parses SSE. | Full control over headers, payloads, cancellation, errors, and compatibility workarounds. | **Good fallback.** It adds parser and protocol code that Balaur must test and maintain. [Dart API](https://api.dart.dev/dart-io/HttpClientResponse-class.html) [HTTP API](https://pub.dev/documentation/http/latest/http/StreamedResponse-class.html) |
| `openai_dart` | Maintained community package from a verified publisher. It lists Android, iOS, Linux, macOS, web, and Windows. | Custom `baseUrl`, Chat Completions, Responses, SSE helpers, tools, structured output, and Realtime support. | **Best current fit.** It gives typed transport without an orchestration framework. [Package](https://pub.dev/packages/openai_dart) [Changelog](https://pub.dev/packages/openai_dart/changelog) |
| `dart_openai` | Established community package. It lists Android, iOS, Linux, macOS, and Windows. Its recent update pace is slower. | Custom global `baseUrl`, Chat Completions streaming, and tools. Some package documentation reports incomplete streaming coverage. | **Second choice.** Global configuration and documented streaming gaps make it less suitable. [Package](https://pub.dev/packages/dart_openai) [Changelog](https://pub.dev/packages/dart_openai/changelog) [Repository](https://github.com/anasfik/openai) |
| LangChain.dart | Community framework with Android, iOS, Linux, macOS, web, and Windows support. Current package pages show a slower release pace. | Streaming, structured output, memory abstractions, tools, `ToolsAgent`, and `AgentExecutor`. Custom OpenAI base URLs are supported. | **Defer.** It adds abstractions Balaur does not need for chat. Reconsider for complex provider-neutral chains. [Core package](https://pub.dev/packages/langchain) [OpenAI package](https://pub.dev/packages/langchain_openai) [Repository](https://github.com/davidmigloz/langchain_dart) |
| Genkit Dart | Official Google project in preview. Dart APIs can change. | Streaming, typed flows, structured output, tools, bounded model turns, agents, sessions, and Shelf deployment. `genkit_openai` supports compatible base URLs. | **Defer.** It is suitable for a future Dart backend, but preview status and scope conflict with KISS. [Dart announcement](https://dart.dev/blog/announcing-genkit-dart-build-full-stack-ai-apps-with-dart-and-flutter) [Core package](https://pub.dev/packages/genkit) [OpenAI plugin](https://pub.dev/packages/genkit_openai) |
| `dart_mcp` | Official Dart-team package from `labs.dart.dev`. It is experimental and supports all six Flutter platforms. | MCP clients and servers, with strong local and standard-input/output use cases. Transport support is still developing. | **Defer.** MCP connects tools and context. It does not replace chat transport or an agent loop. [Package](https://pub.dev/packages/dart_mcp) [Source](https://github.com/dart-lang/ai/tree/main/pkgs/dart_mcp) |

### 3. Official OpenAI language support

OpenAI has no official Dart or Flutter API SDK. Its SDK page lists Dart only among community libraries. [OpenAI SDKs](https://platform.openai.com/docs/libraries)

The official OpenAI Agents SDK supports Python and TypeScript. It does not support Dart. [Python Agents SDK](https://openai.github.io/openai-agents-python/) [TypeScript Agents SDK](https://openai.github.io/openai-agents-js/)

Therefore, Balaur has three practical choices:

- Use a community Dart client.
- Call the REST API directly.
- Put an official SDK or Agents SDK on a Python or TypeScript backend.

The backend choice is useful only when Balaur needs centralized tools, tracing, guardrails, or multi-agent orchestration.

### 4. Client execution, backend execution, and keys

A Flutter client can call the custom endpoint directly. This design gives low latency and simple streaming.

A distributed client cannot protect a permanent service secret. OpenAI says not to deploy secret keys in mobile applications or browsers. It recommends routing requests through a backend. [OpenAI key safety](https://help.openai.com/en/articles/5112595-best-practices-for-api-key-safety)

Use one of these security models:

- **Service-owned key:** Use an authenticated backend proxy. Store the upstream key in a secret manager or environment variable.
- **User-owned key:** Store it with platform secure storage. State clearly that a determined device owner can still extract it.
- **Trusted local endpoint:** Allow direct access when no reusable secret leaves the device.

Do not use Firebase App Check as a general solution for Balaur's custom endpoint. Firebase AI Logic uses App Check for its own supported service path. [Firebase production checklist](https://firebase.google.com/docs/ai-logic/production-checklist)

### 5. Streaming and protocol design

Chat Completions uses data-only SSE when `stream: true`. Text arrives in `choices[].delta.content`, and `[DONE]` ends the stream. [OpenAI streaming](https://platform.openai.com/docs/guides/streaming-responses) [Chat API](https://platform.openai.com/docs/api-reference/chat/create)

If Balaur uses direct HTTP, its parser must handle event boundaries, multiple `data:` lines, heartbeats, errors, and cancellation. It must not treat each byte chunk as one event.

Prefer typed gateway events instead of exposing raw strings:

- `textDelta`
- `toolCallDelta`
- `usage`
- `completed`
- `failed`

This design permits later tool support without changing the user interface contract.

### 6. Tools, structured output, and agent loops

Chat Completions supports function tools. The model requests a function, but the application executes it and returns a tool message. [OpenAI function calling](https://platform.openai.com/docs/guides/function-calling)

Structured Outputs can constrain final JSON with a JSON Schema. Model and endpoint support still require verification. [OpenAI Structured Outputs](https://platform.openai.com/docs/guides/structured-outputs)

`openai_dart` represents tool calls and structured formats, but Balaur must execute tools itself. [Package](https://pub.dev/packages/openai_dart)

When domain tools arrive, add a bounded loop inside the gateway or backend:

1. Send the conversation and allowed tool schemas.
2. Accumulate the complete streamed tool call.
3. Validate the tool name and arguments.
4. Apply authorization and confirmation rules.
5. Execute the tool.
6. Append the tool result.
7. Repeat until final text or the turn limit.

Do not add an autonomous loop before Balaur has tools. Use Genkit or LangChain.dart only when this small loop becomes hard to maintain.

### 7. Conversation persistence

Neither a chat UI nor a model client is the durable source of truth. Store conversations in an app-owned repository.

Store at least these fields:

- Conversation identifier and timestamps.
- Ordered message identifier and role.
- Text and attachment references.
- Tool call and tool result records.
- Model and endpoint metadata.
- Delivery state, error state, and completion state.

Persist completed turns. Keep partial streamed text in transient state until completion. Restore model messages from stored records when a conversation opens.

Flutter AI Toolkit only supplies message serialization and provider history. The application still owns durable storage. [Feature integration](https://docs.flutter.dev/ai/ai-toolkit/feature-integration)

Firebase `ChatSession` keeps successful history in memory. Applications restore history through `startChat(history: ...)`. [Firebase chat](https://firebase.google.com/docs/ai-logic/chat) [ChatSession API](https://pub.dev/documentation/firebase_ai/latest/firebase_ai/ChatSession-class.html)

## Sources

### Kept

- [Flutter AI Toolkit](https://docs.flutter.dev/ai/ai-toolkit) — official Flutter scope and platform information.
- [Firebase AI Logic](https://firebase.google.com/docs/ai-logic) — official Firebase capabilities and security model.
- [Google Gen AI libraries](https://ai.google.dev/gemini-api/docs/libraries) — official SDK language and deprecation status.
- [OpenAI SDKs](https://platform.openai.com/docs/libraries) — official and community language classification.
- [OpenAI API guides](https://platform.openai.com/docs/guides/function-calling) — protocol behavior for tools and streaming.
- [Genkit Dart](https://github.com/genkit-ai/genkit-dart) — official Dart framework source and status.
- [Dart MCP](https://github.com/dart-lang/ai/tree/main/pkgs/dart_mcp) — official experimental MCP implementation.
- [Pub.dev package pages](https://pub.dev/) — package ownership, platforms, versions, and published capabilities.

### Dropped

- Blogs, comparison sites, and tutorial articles — they are not primary sources.
- Forum posts — official pages and repositories supplied stronger evidence.
- Unrelated agent packages — they did not have official or requested framework relevance.

## Gaps and residual risks

- Balaur's custom endpoint was not available for protocol tests.
- Provider compatibility can differ for SSE termination, tool deltas, usage, errors, and JSON Schema enforcement.
- Package versions and preview APIs can change after this check.
- Linux secret storage and network behavior need integration tests on Balaur's supported distributions.
- A backend adds deployment work, but it is necessary for a service-owned permanent key.

## Acceptance evidence

- **Finding severity:** No blocker exists for basic chat. Linux excludes `flutter_ai_toolkit` and `firebase_ai` as full Balaur solutions.
- **Changed file:** `/home/alex/Work/balaur/research.md`.
- **Tests:** No tests were added because this task changed documentation only.
- **Manual note:** Validate the custom endpoint before selecting optional tool or structured-output features.

```acceptance-report
{
  "criteriaSatisfied": [
    {
      "id": "criterion-1",
      "status": "satisfied",
      "evidence": "research.md contains cited findings, option status, platform support, recommendations, and severity for Linux incompatibilities."
    }
  ],
  "changedFiles": [
    "/home/alex/Work/balaur/research.md"
  ],
  "testsAddedOrUpdated": [],
  "commandsRun": [
    {
      "command": "Focused primary-source web research and source-page review",
      "result": "passed",
      "summary": "Reviewed official documentation, repositories, source pages, and pub.dev package pages."
    },
    {
      "command": "Automated tests",
      "result": "not-run",
      "summary": "The task changed only a Markdown research report."
    }
  ],
  "validationOutput": [
    "The report covers every requested option and architectural concern.",
    "The report recommends an Android, iOS, and Linux architecture for a custom OpenAI-compatible endpoint.",
    "Only /home/alex/Work/balaur/research.md was written."
  ],
  "residualRisks": [
    "The custom endpoint needs protocol compatibility tests.",
    "Current package and preview-framework status can change."
  ],
  "noStagedFiles": true,
  "diffSummary": "Added one cited Markdown research report. No source files changed.",
  "reviewFindings": [
    "no blockers",
    "medium: flutter_ai_toolkit and firebase_ai do not list Linux support",
    "medium: service-owned API keys require a backend proxy"
  ],
  "manualNotes": "The authoritative runtime path overrides the docs/research/flutter-ai-options.md path in the task text."
}
```
