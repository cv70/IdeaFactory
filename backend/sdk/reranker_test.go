package sdk

import (
	"log/slog"
	"testing"
)

func TestRerank(t *testing.T) {
	reranker := AnythingRerankerClient{
		Model: "Qwen3-Reranker-0.6B",
		URL:   "http://10.5.55.55:18083/score",
	}
	items, err := reranker.Rerank("类比推理", []string{
		`国家发展：民族进步：社会和谐
A:农村美丽：民风淳朴：农家幸福
B:榜样力量：鼓舞人心：引导征程
C:智慧增长：信心增强：干劲增大
D:传统习惯：文化底蕴：历史传承`,
		`手绘动画的运动表演、镜头组织与透视控制实际上高度依赖作画者的技术与经验：同一场景的复杂调度，三维可以在立体场景中摆机位、走摄像，二维则必须经由原画-中割-合成层层推进，这就导致二维动画不仅返工成本高，同时节奏还难以把控。网络小说式的超长叙事更会把二维的周期与预算进一步拉紧。因此我们看到的大部分二维内容都来自于“原创”或“漫改”，除非有着充足时间，或者有着既有分镜。若是希望做出某些“奇观内容”，则更需要风格适配、经验丰富、协作得当、经费充足的制作团队。换言之，______。
填入画横线部分最恰当的( )
A:手绘动画应以工业支撑为骨，以艺术突破为魂
B:生产效率与空间表现共同构成二维动画的弱势
C:二维动画面临的问题在于流程之难与结构之困
D:二维动画的工序之冗余、条件之苛刻难以想象`}, &RerankConfig{
		Instruction: `Find relevant exam questions from the candidate set based on the query`,
	})
	if err != nil {
		slog.Error("rerank error", slog.Any("err", err))
	}
	slog.Info("result:", slog.Any("items", items))
}
