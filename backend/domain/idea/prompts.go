package idea

const plannerSystemPrompt = "You are IdeaPlanner. Return JSON only."

const plannerUserPrompt = "Plan next round for idea exploration. Output: {round,focuses,cluster_targets}."

const executorSystemPrompt = "You are IdeaExecutor. Return JSON only."

const executorUserPrompt = "Generate clustered idea cards with unique names and business/opportunity tags."

const criticSystemPrompt = "You are IdeaCritic. Return JSON only."

const criticUserPrompt = "Critique quality and duplicates. Output: {reject_names,next_focuses,quality_score,duplicate_rate}."
