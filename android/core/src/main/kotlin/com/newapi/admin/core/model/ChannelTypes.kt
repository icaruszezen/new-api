package com.newapi.admin.core.model

data class ChannelType(val id: Int, val name: String)

/** Channel type ids from `constant/channel.go`, labelled like the web (`web/src/features/channels/constants.ts`). */
object ChannelTypes {
    private val names: Map<Int, String> = mapOf(
        0 to "Unknown",
        1 to "OpenAI",
        2 to "MjProxy",
        3 to "Azure",
        4 to "Ollama",
        5 to "MjProxyPlus",
        6 to "OpenAIMax",
        7 to "OhMyGPT",
        8 to "Custom",
        9 to "AILS",
        10 to "AIProxy",
        11 to "PaLM",
        12 to "API2GPT",
        13 to "AIGC2D",
        14 to "Anthropic",
        15 to "Baidu",
        16 to "Zhipu",
        17 to "Ali",
        18 to "Xunfei",
        19 to "360",
        20 to "OpenRouter",
        21 to "AIProxyLibrary",
        22 to "FastGPT",
        23 to "Tencent",
        24 to "Gemini",
        25 to "Moonshot",
        26 to "Zhipu V4",
        27 to "Perplexity",
        31 to "LingYiWanWu",
        33 to "AWS",
        34 to "Cohere",
        35 to "MiniMax",
        36 to "SunoAPI",
        37 to "Dify",
        38 to "Jina",
        39 to "Cloudflare",
        40 to "SiliconFlow",
        41 to "Vertex AI",
        42 to "Mistral",
        43 to "DeepSeek",
        44 to "MokaAI",
        45 to "VolcEngine",
        46 to "Baidu V2",
        47 to "Xinference",
        48 to "xAI",
        49 to "Coze",
        50 to "Kling",
        51 to "Jimeng",
        52 to "Vidu",
        53 to "Submodel",
        54 to "DoubaoVideo",
        55 to "Sora",
        56 to "Replicate",
        57 to "ChatGPT Subscription (Codex)",
        58 to "Advanced Custom",
        59 to "Sub2API",
        60 to "New API",
    )

    /** Types the web hides from its picker; existing channels may still use them. */
    private val hidden = setOf(6, 9, 10, 11, 12, 13, 21)

    /** Web picker order. */
    private val pickerOrder = listOf(
        1, 14, 33, 24, 43, 3, 41, 48, 60, 58, 42, 34, 20, 4, 40, 27, 25, 17, 26, 15, 46, 23, 18, 45, 31, 35,
        49, 19, 47, 37, 38, 39, 11, 8, 57, 59, 22, 21, 44, 2, 5, 36, 50, 51, 52, 53, 54, 55, 56,
    )

    /** Types offered when creating a channel. */
    val selectable: List<ChannelType> = pickerOrder.filter { it !in hidden }.map { ChannelType(it, names.getValue(it)) }

    fun nameOf(type: Int): String = names[type] ?: "#$type"

    /** Types whose upstream balance can be queried (`controller/channel-billing.go`). */
    val balanceSupported: Set<Int> = setOf(1, 8, 10, 12, 13, 20, 25, 40, 43, 58)

    /** Types the single-channel test does not support. */
    val testUnsupported: Set<Int> = setOf(2, 5, 36, 50, 51, 52, 54)

    /** Types whose base URL the server requires (60) or that are unusable without one (3, 8, 58, 59). */
    val baseUrlRequired: Set<Int> = setOf(3, 8, 58, 59, 60)
}
