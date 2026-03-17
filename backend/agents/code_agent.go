package agents

import (
	"backend/agentools"
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/adk/multiagent/deep/params"
	"github.com/cloudwego/eino-examples/adk/multiagent/deep/utils"
)

// NewCodeAgent 创建一个专门处理Excel文件的代码代理。
// 该代理接收明确的任务，生成Python代码并执行它来处理Excel文件。
// 它利用pandas进行数据分析和操作，matplotlib进行绘图和可视化，
// 以及openpyxl读写Excel文件。
// 参数:
//   - ctx: 上下文，用于控制取消和传递请求范围值。
//   - model: 用于工具调用的聊天模型。
//   - operator: 命令行操作符，用于执行系统命令。
//
// 返回:
//   - adk.Agent: 代码代理实例。
//   - error: 如果创建过程中发生错误。
func NewCodeAgent(ctx context.Context, model model.ToolCallingChatModel, operator commandline.Operator) (adk.Agent, error) {
	// 预处理函数：修复JSON格式
	preprocess := []agentools.ToolRequestPreprocess{agentools.ToolRequestRepairJSON}

	// 创建聊天模型代理配置
	ca, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name: "CodeAgent",
		Description: `This sub-agent is a code agent specialized in handling Excel files. 
It receives a clear task and accomplish the task by generating Python code and execute it. 
The agent leverages pandas for data analysis and manipulation, matplotlib for plotting and visualization, and openpyxl for reading and writing Excel files. 
The React agent should invoke this sub-agent whenever stepwise Python coding for Excel file operations is required, ensuring precise and efficient task execution.`,
		Instruction: `You are a code agent. Your workflow is as follows:
1. You will be given a clear task to handle Excel files.
2. You should analyse the task and use right tools to help coding.
3. You should write python code to finish the task.
4. You are preferred to write code execution result to another file for further usages. 

You are in a react mode, and you should use the following libraries to help you finish the task:
- pandas: for data analysis and manipulation
- matplotlib: for plotting and visualization
- openpyxl: for reading and writing Excel files

Notice:
1. Tool Calls argument must be a valid json.
2. Tool Calls argument should do not contains invalid suffix like ']<|FunctionCallEnd|>'. 
`,
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					// Bash工具：执行系统命令，带有JSON修复和文件后处理
					agentools.NewWrapTool(agentools.NewBashTool(operator), preprocess, []agentools.ToolResponsePostprocess{agentools.FilePostProcess}),
					// Tree工具：显示目录结构，带有JSON修复
					agentools.NewWrapTool(agentools.NewTreeTool(operator), preprocess, nil),
					// EditFile工具：编辑文件，带有JSON修复和文件编辑后处理
					agentools.NewWrapTool(agentools.NewEditFileTool(operator), preprocess, []agentools.ToolResponsePostprocess{agentools.EditFilePostProcess}),
					// ReadFile工具：读取文件，带有JSON修复
					agentools.NewWrapTool(agentools.NewReadFileTool(operator), preprocess, nil), // TODO: 添加压缩后处理
					// PythonRunner工具：执行Python代码，带有JSON修复和文件后处理
					agentools.NewWrapTool(agentools.NewPythonRunnerTool(operator), preprocess, []agentools.ToolResponsePostprocess{agentools.FilePostProcess}),
				},
			},
		},
		GenModelInput: func(ctx context.Context, instruction string, input *adk.AgentInput) ([]adk.Message, error) {
			// 从上下文获取工作目录
			wd, ok := params.GetTypedContextParams[string](ctx, params.WorkDirSessionKey)
			if !ok {
				return nil, fmt.Errorf("工作目录未找到")
			}

			// 创建提示模板
			tpl := prompt.FromMessages(schema.GoTemplate,
				schema.SystemMessage(instruction),
				schema.UserMessage(`WorkingDirectory: {{ .working_dir }}
UserQuery: {{ .user_query }}
CurrentTime: {{ .current_time }}
`))

			// 格式化模板参数
			msgs, err := tpl.Format(ctx, map[string]any{
				"working_dir":  wd,
				"user_query":   utils.FormatInput(input.Messages),
				"current_time": utils.GetCurrentTime(),
			})
			if err != nil {
				return nil, err
			}

			return msgs, nil
		},
		OutputKey:     "",
		MaxIterations: 1000,
	})
	if err != nil {
		return nil, err
	}

	return ca, nil
}
