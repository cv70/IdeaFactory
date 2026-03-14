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

func NewCodeAgent(ctx context.Context, model model.ToolCallingChatModel, operator commandline.Operator) (adk.Agent, error) {
	preprocess := []agentools.ToolRequestPreprocess{agentools.ToolRequestRepairJSON}

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
					agentools.NewWrapTool(agentools.NewBashTool(operator), preprocess, []agentools.ToolResponsePostprocess{agentools.FilePostProcess}),
					agentools.NewWrapTool(agentools.NewTreeTool(operator), preprocess, nil),
					agentools.NewWrapTool(agentools.NewEditFileTool(operator), preprocess, []agentools.ToolResponsePostprocess{agentools.EditFilePostProcess}),
					agentools.NewWrapTool(agentools.NewReadFileTool(operator), preprocess, nil), // TODO: compress post process
					agentools.NewWrapTool(agentools.NewPythonRunnerTool(operator), preprocess, []agentools.ToolResponsePostprocess{agentools.FilePostProcess}),
				},
			},
		},
		GenModelInput: func(ctx context.Context, instruction string, input *adk.AgentInput) ([]adk.Message, error) {
			wd, ok := params.GetTypedContextParams[string](ctx, params.WorkDirSessionKey)
			if !ok {
				return nil, fmt.Errorf("work dir not found")
			}

			tpl := prompt.FromMessages(schema.GoTemplate,
				schema.SystemMessage(instruction),
				schema.UserMessage(`WorkingDirectory: {{ .working_dir }}
UserQuery: {{ .user_query }}
CurrentTime: {{ .current_time }}
`))

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
