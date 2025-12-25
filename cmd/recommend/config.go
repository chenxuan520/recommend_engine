package main

import (
	"flag"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// LLMGlobalConfig 对应 configs/llm.yaml
type LLMGlobalConfig struct {
	LLMs map[string]struct {
		ChatEndpoint string `yaml:"chat_endpoint"` // 完整的 API 地址
		APIKey       string `yaml:"api_key"`
		Model        string `yaml:"model"`
	} `yaml:"llms"`
}

// ServerConfig 对应 configs/server.yaml
type ServerConfig struct {
	Server struct {
		Port  string `yaml:"port"`
		Debug bool   `yaml:"debug"`
	} `yaml:"server"`
	Paths struct {
		Users     string `yaml:"users"`
		Pipelines string `yaml:"pipelines"`
		LLM       string `yaml:"llm"`
		History   string `yaml:"history"`
	} `yaml:"paths"`
}

func loadLLMConfig(path string) (*LLMGlobalConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg LLMGlobalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func loadServerConfig(path string) (*ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg ServerConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// InitServerConfig 初始化服务器配置，优先级：命令行参数 > 配置文件 > 默认值
func InitServerConfig() *ServerConfig {
	// 命令行参数
	// 将默认值设置为空字符串，以便优先使用配置文件中的值
	configPath := flag.String("config", "configs/server.yaml", "Path to server config file")
	portFlag := flag.String("port", "", "Server port")
	debugFlag := flag.Bool("debug", false, "Enable debug logging")
	userConfigPathFlag := flag.String("users", "", "Path to users.yaml")
	pipelineConfigPathFlag := flag.String("pipelines", "", "Path to pipelines.json")
	llmConfigPathFlag := flag.String("llm", "", "Path to llm.yaml")
	historyPathFlag := flag.String("history", "", "Path to history.jsonl")
	flag.Parse()

	// 1. 初始化默认值
	serverCfg := &ServerConfig{}
	serverCfg.Server.Port = "8080"
	serverCfg.Server.Debug = false
	serverCfg.Paths.Users = "configs/users.yaml"
	serverCfg.Paths.Pipelines = "configs/pipelines.json"
	serverCfg.Paths.LLM = "configs/llm.yaml"
	serverCfg.Paths.History = "data/history.jsonl"

	// 2. 尝试加载配置文件
	if loadedCfg, err := loadServerConfig(*configPath); err == nil {
		// 如果文件存在且加载成功，覆盖默认值
		if loadedCfg.Server.Port != "" {
			serverCfg.Server.Port = loadedCfg.Server.Port
		}
		// Debug 默认为 false，如果配置文件里显式设置了 true 则覆盖
		if loadedCfg.Server.Debug {
			serverCfg.Server.Debug = true
		}
		if loadedCfg.Paths.Users != "" {
			serverCfg.Paths.Users = loadedCfg.Paths.Users
		}
		if loadedCfg.Paths.Pipelines != "" {
			serverCfg.Paths.Pipelines = loadedCfg.Paths.Pipelines
		}
		if loadedCfg.Paths.LLM != "" {
			serverCfg.Paths.LLM = loadedCfg.Paths.LLM
		}
		if loadedCfg.Paths.History != "" {
			serverCfg.Paths.History = loadedCfg.Paths.History
		}
	} else {
		// 只有当用户显式指定了配置文件但加载失败时才报错，
		// 或者如果默认文件不存在，我们就不报错，直接使用硬编码默认值
		// 这里简化处理：只打印日志
		log.Printf("Info: Could not load config file '%s': %v. Using defaults or flags.", *configPath, err)
	}

	// 3. 应用命令行参数 (优先级最高)
	if *portFlag != "" {
		serverCfg.Server.Port = *portFlag
	}
	if *debugFlag {
		serverCfg.Server.Debug = true
	}
	if *userConfigPathFlag != "" {
		serverCfg.Paths.Users = *userConfigPathFlag
	}
	if *pipelineConfigPathFlag != "" {
		serverCfg.Paths.Pipelines = *pipelineConfigPathFlag
	}
	if *llmConfigPathFlag != "" {
		serverCfg.Paths.LLM = *llmConfigPathFlag
	}
	if *historyPathFlag != "" {
		serverCfg.Paths.History = *historyPathFlag
	}
	
	return serverCfg
}
