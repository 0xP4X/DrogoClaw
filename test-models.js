fetch("https://openrouter.ai/api/v1/models")
  .then(r => r.json())
  .then(d => {
    const models = d.data.map(m => m.id);
    console.log("Claude models:", models.filter(m => m.includes("claude-3")));
  });
