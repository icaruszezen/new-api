package model

import (
	"slices"
	"strings"
	"sync"
)

// 简化的供应商映射规则
var defaultVendorRules = map[string]string{
	"gpt":      "OpenAI",
	"dall-e":   "OpenAI",
	"whisper":  "OpenAI",
	"o1":       "OpenAI",
	"o3":       "OpenAI",
	"claude":   "Anthropic",
	"gemini":   "Google",
	"moonshot": "Moonshot",
	"kimi":     "Moonshot",
	"chatglm":  "智谱",
	"glm-":     "智谱",
	"qwen":     "阿里巴巴",
	"deepseek": "DeepSeek",
	"abab":     "MiniMax",
	"minimax":  "MiniMax",
	"ernie":    "百度",
	"spark":    "讯飞",
	"hunyuan":  "腾讯",
	"command":  "Cohere",
	"@cf/":     "Cloudflare",
	"360":      "360",
	"yi":       "零一万物",
	"jina":     "Jina",
	"mistral":  "Mistral",
	"grok":     "xAI",
	"llama":    "Meta",
	"doubao":   "字节跳动",
	"kling":    "快手",
	"jimeng":   "即梦",
	"vidu":     "Vidu",
}

// 供应商默认图标映射
var defaultVendorIcons = map[string]string{
	"OpenAI":     "OpenAI",
	"Anthropic":  "Claude.Color",
	"Google":     "Gemini.Color",
	"Moonshot":   "Moonshot",
	"智谱":         "Zhipu.Color",
	"阿里巴巴":       "Qwen.Color",
	"DeepSeek":   "DeepSeek.Color",
	"MiniMax":    "Minimax.Color",
	"百度":         "Wenxin.Color",
	"讯飞":         "Spark.Color",
	"腾讯":         "Hunyuan.Color",
	"Cohere":     "Cohere.Color",
	"Cloudflare": "Cloudflare.Color",
	"360":        "Ai360.Color",
	"零一万物":       "Yi.Color",
	"Jina":       "Jina",
	"Mistral":    "Mistral.Color",
	"xAI":        "XAI",
	"Meta":       "Ollama",
	"字节跳动":       "Doubao.Color",
	"快手":         "Kling.Color",
	"即梦":         "Jimeng.Color",
	"Vidu":       "Vidu",
	"微软":         "AzureAI",
	"Microsoft":  "AzureAI",
	"Azure":      "AzureAI",
}

// initDefaultVendorMapping 简化的默认供应商映射
func initDefaultVendorMapping(metaMap map[string]*Model, vendorMap map[int]*Vendor, enableAbilities []AbilityWithChannel) {
	patterns := sortedVendorRulePatterns()
	for _, ability := range enableAbilities {
		modelName := ability.Model
		if _, exists := metaMap[modelName]; exists {
			continue
		}

		// 匹配供应商
		vendorID := 0
		modelLower := strings.ToLower(modelName)
		for _, pattern := range patterns {
			vendorName := defaultVendorRules[pattern]
			if strings.Contains(modelLower, pattern) {
				vendorID = getDisplayVendor(vendorName, vendorMap)
				break
			}
		}

		// 创建模型元数据
		metaMap[modelName] = &Model{
			ModelName: modelName,
			VendorID:  vendorID,
			Status:    1,
			NameRule:  NameRuleExact,
		}
	}
}

// Default vendor entries are presentation data. Reading pricing must never
// recreate a deleted/merged database record.
var defaultVendorDisplayIDs = map[string]int{
	"360":        -1001,
	"Anthropic":  -1002,
	"Cloudflare": -1003,
	"Cohere":     -1004,
	"DeepSeek":   -1005,
	"Google":     -1006,
	"Jina":       -1007,
	"Meta":       -1008,
	"MiniMax":    -1009,
	"Mistral":    -1010,
	"Moonshot":   -1011,
	"OpenAI":     -1012,
	"Vidu":       -1013,
	"xAI":        -1014,
	"即梦":         -1015,
	"字节跳动":       -1016,
	"快手":         -1017,
	"智谱":         -1018,
	"百度":         -1019,
	"腾讯":         -1020,
	"讯飞":         -1021,
	"阿里巴巴":       -1022,
	"零一万物":       -1023,
}

func getDisplayVendor(vendorName string, vendorMap map[int]*Vendor) int {
	for id, vendor := range vendorMap {
		if strings.EqualFold(vendor.Name, vendorName) {
			return id
		}
	}
	id := defaultVendorDisplayIDs[vendorName]
	if id == 0 {
		return 0
	}
	vendorMap[id] = &Vendor{Id: id, Name: vendorName, Status: 1, Icon: getDefaultVendorIcon(vendorName)}
	return id
}

// 获取供应商默认图标
func getDefaultVendorIcon(vendorName string) string {
	if icon, exists := defaultVendorIcons[vendorName]; exists {
		return icon
	}
	return ""
}

// ResolveModelIconKey 推导模型对应的 @lobehub/icons 图标名。
// 优先级：模型自定义图标 > 模型关联供应商图标 > 按模型名匹配的默认供应商图标。
// 未能识别时返回空字符串，由前端回退到首字母头像。
func ResolveModelIconKey(modelName string) string {
	trimmed := strings.TrimSpace(modelName)
	if trimmed == "" {
		return ""
	}

	var meta Model
	if err := DB.Where("model_name = ?", trimmed).First(&meta).Error; err == nil {
		if meta.Icon != "" {
			return meta.Icon
		}
		if meta.VendorID > 0 {
			var vendor Vendor
			if err := DB.First(&vendor, meta.VendorID).Error; err == nil && vendor.Icon != "" {
				return vendor.Icon
			}
		}
	}

	modelLower := strings.ToLower(trimmed)
	for _, pattern := range sortedVendorRulePatterns() {
		if strings.Contains(modelLower, pattern) {
			return getDefaultVendorIcon(defaultVendorRules[pattern])
		}
	}
	return ""
}

// sortedVendorRulePatterns 让匹配顺序稳定且偏向更具体的规则：
// 先按模式长度倒序，长度相同时按字典序，避免 map 遍历顺序导致图标随机跳变。
func sortedVendorRulePatterns() []string {
	vendorRulePatternsOnce.Do(func() {
		vendorRulePatterns = make([]string, 0, len(defaultVendorRules))
		for pattern := range defaultVendorRules {
			vendorRulePatterns = append(vendorRulePatterns, pattern)
		}
		slices.SortFunc(vendorRulePatterns, func(a, b string) int {
			if len(a) != len(b) {
				return len(b) - len(a)
			}
			return strings.Compare(a, b)
		})
	})
	return vendorRulePatterns
}

var (
	vendorRulePatterns     []string
	vendorRulePatternsOnce sync.Once
)
