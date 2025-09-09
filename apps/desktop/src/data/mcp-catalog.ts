export type CatalogItem = {
  slug: string;
  name: string;
  category:
    | "Reasoning"
    | "UI / Frontend"
    | "UI / Frontend Docs"
    | "Hosting / Infra"
    | "Cloud"
    | "Cloud / DB"
    | "Infra as Code"
    | "Containers"
    | "Dev"
    | "Design"
    | "Comms"
    | "Email"
    | "Docs"
    | "Calendar"
    | "Notes"
    | "Social"
    | "Messaging"
    | "Social / Newsletters"
    | "Monitoring"
    | "Security / Recon"
    | "Security / Net";
  description: string;
  repoUrl?: string;
  docsUrl?: string;
  install?: { type: "npm" | "git" | "pip" | "docker-image" | "docker-compose"; uri: string };
  remote?: { apiEndpoint: string; provider: string; authType?: "api_key" | "oauth2" | "basic" };
  configExample?: string;
  tags?: string[];
};

export const CATALOG: CatalogItem[] = [
  { slug: "sequential-thinking", name: "Sequential Thinking", category: "Reasoning", description: "Step-by-step structured reasoning.", repoUrl: "https://www.npmjs.com/package/@modelcontextprotocol/sequentialthinking", install: { type: "npm", uri: "@modelcontextprotocol/sequentialthinking" }, configExample: '{ "mcpServers": { "sequential-thinking": { "command": "npx", "args": ["-y","@modelcontextprotocol/sequentialthinking"] } } }', tags: ["reasoning"] },
  { slug: "crash", name: "CRASH (Cascaded Reasoning)", category: "Reasoning", description: "Confidence tracking, branching, adaptive steps.", repoUrl: "https://github.com/modelcontextprotocol/crash-mcp", install: { type: "npm", uri: "crash-mcp" }, configExample: '{ "mcpServers": { "crash": { "command":"npx","args":["-y","crash-mcp"] } } }' },
  { slug: "reasoner", name: "MCP Reasoner (Beam/MCTS)", category: "Reasoning", description: "Beam search & MCTS reasoning.", repoUrl: "https://glama.ai/mcp/Reasoner", install: { type: "npm", uri: "mcp-reasoner" }, configExample: '{ "mcpServers": { "reasoner": { "command":"npx","args":["-y","mcp-reasoner"] } } }' },
  { slug: "mcts", name: "MCTS MCP", category: "Reasoning", description: "Monte Carlo Tree Search.", repoUrl: "https://github.com/search?q=MCTS+MCP+server", install: { type: "npm", uri: "mcts-mcp" }, configExample: '{ "mcpServers": { "mcts": { "command":"npx","args":["-y","mcts-mcp"] } } }' },
  { slug: "got", name: "Graph-of-Thoughts", category: "Reasoning", description: "Graph reasoning (Neo4j optional).", repoUrl: "https://github.com/saptadey/got-mcp", install: { type: "npm", uri: "got-mcp" }, configExample: '{ "mcpServers": { "got": { "command":"npx","args":["-y","got-mcp"] } } }' },
  { slug: "dre", name: "Deliberate Reasoning Engine (DRE)", category: "Reasoning", description: "DAG / thought-graph reasoning.", repoUrl: "https://glama.ai/mcp/DRE", install: { type: "npm", uri: "dre-mcp" }, configExample: '{ "mcpServers": { "dre": { "command":"npx","args":["-y","dre-mcp"] } } }' },
  { slug: "branch-thinking", name: "Branch Thinking", category: "Reasoning", description: "Manage multiple reasoning branches.", repoUrl: "https://glama.ai/mcp/BranchThinking", install: { type: "npm", uri: "branch-thinking-mcp" }, configExample: '{ "mcpServers": { "branch-thinking": { "command":"npx","args":["-y","branch-thinking-mcp"] } } }' },
  { slug: "cot", name: "Chain-of-Thought (beverm2391)", category: "Reasoning", description: "Exposes CoT tokens.", repoUrl: "https://github.com/beverm2391/cot-mcp", install: { type: "npm", uri: "cot-mcp" }, configExample: '{ "mcpServers": { "cot": { "command":"npx","args":["-y","cot-mcp"] } } }' },
  { slug: "cot-task", name: "CoT Task Manager", category: "Reasoning", description: "Task decomposition with CoT.", repoUrl: "https://github.com/liorfranko/cot-task-manager", install: { type: "npm", uri: "cot-task-mcp" }, configExample: '{ "mcpServers": { "cot-task": { "command":"npx","args":["-y","cot-task-mcp"] } } }' },
  { slug: "planner", name: "Software Planner", category: "Reasoning", description: "Software project planning.", repoUrl: "https://github.com/modelcontextprotocol/planner-mcp", install: { type: "npm", uri: "planner-mcp" }, configExample: '{ "mcpServers": { "planner": { "command":"npx","args":["-y","planner-mcp"] } } }' },
  { slug: "memory", name: "Memory MCP", category: "Reasoning", description: "Persistent knowledge graph memory.", repoUrl: "https://github.com/modelcontextprotocol/memory-mcp", install: { type: "npm", uri: "memory-mcp" }, configExample: '{ "mcpServers": { "memory": { "command":"npx","args":["-y","memory-mcp"] } } }' },
  { slug: "mindmap", name: "Mindmap MCP", category: "Reasoning", description: "Convert ideas into mindmaps.", repoUrl: "https://github.com/modelcontextprotocol/mindmap-mcp", install: { type: "npm", uri: "mindmap-mcp" }, configExample: '{ "mcpServers": { "mindmap": { "command":"npx","args":["-y","mindmap-mcp"] } } }' },
  { slug: "context-crystallizer", name: "Context Crystallizer", category: "Reasoning", description: "Distill docs/repos into structured knowledge.", repoUrl: "https://github.com/modelcontextprotocol/context-crystallizer-mcp", install: { type: "npm", uri: "context-crystallizer-mcp" }, configExample: '{ "mcpServers": { "context-crystallizer": { "command":"npx","args":["-y","context-crystallizer-mcp"] } } }' },
  { slug: "shadcn-ui", name: "Shadcn UI MCP", category: "UI / Frontend", description: "Search/install shadcn/ui components.", repoUrl: "https://github.com/Jpisnice/shadcn-ui-mcp-server", install: { type: "npm", uri: "shadcn-ui-mcp-server" }, configExample: '{ "mcpServers": { "shadcn-ui": { "command":"npx","args":["-y","shadcn-ui-mcp-server"] } } }' },
  { slug: "assistant-ui-docs", name: "assistant-ui Docs MCP", category: "UI / Frontend Docs", description: "Assistant-ui docs/examples in IDE.", repoUrl: "https://github.com/assistant-ui/mcp-docs", install: { type: "npm", uri: "@assistant-ui/mcp-docs-server" }, configExample: '{ "mcpServers": { "assistant-ui-docs": { "command":"npx","args":["-y","@assistant-ui/mcp-docs-server"] } } }' },
  { slug: "render", name: "Render MCP", category: "Hosting / Infra", description: "Manage Render services/deploys.", repoUrl: "https://github.com/render-oss/render-mcp-server", remote: { apiEndpoint: "https://mcp.render.com/mcp", provider: "Render", authType: "api_key" }, install: { type: "npm", uri: "render-mcp-server" }, configExample: '{ "mcpServers": { "render": { "url": "https://mcp.render.com/mcp", "headers": { "Authorization": "Bearer <RENDER_API_KEY>" } } } }' },
  { slug: "flyio", name: "Fly.io MCP", category: "Hosting / Infra", description: "Manage Fly.io apps via flyctl.", repoUrl: "https://github.com/superfly/flymcp", install: { type: "npm", uri: "flymcp" }, configExample: '{ "mcpServers": { "flyio": { "command":"fly","args":["mcp","server"] } } }' },
  { slug: "aws-ccapi", name: "AWS CCAPI MCP", category: "Cloud", description: "Natural language AWS resource mgmt.", repoUrl: "https://github.com/awslabs/mcp", install: { type: "npm", uri: "@awslabs/ccapi-mcp-server" }, configExample: '{ "mcpServers": { "aws-ccapi": { "command":"npx","args":["-y","@awslabs/ccapi-mcp-server"] } } }' },
  { slug: "aws-serverless", name: "AWS Serverless MCP", category: "Cloud", description: "Lambda/serverless guidance.", repoUrl: "https://github.com/awslabs/mcp-serverless", install: { type: "npm", uri: "aws-serverless-mcp" }, configExample: '{ "mcpServers": { "aws-serverless": { "command":"npx","args":["-y","aws-serverless-mcp"] } } }' },
  { slug: "gcp", name: "GCP MCP", category: "Cloud", description: "Google Cloud Platform.", repoUrl: "https://github.com/devinschumacher/gcp-mcp", install: { type: "npm", uri: "gcp-mcp" }, configExample: '{ "mcpServers": { "gcp": { "command":"npx","args":["-y","gcp-mcp"] } } }' },
  { slug: "azure", name: "Azure MCP", category: "Cloud", description: "Manage Azure/DevOps.", repoUrl: "https://github.com/devinschumacher/azure-mcp", install: { type: "npm", uri: "azure-mcp" }, configExample: '{ "mcpServers": { "azure": { "command":"npx","args":["-y","azure-mcp"] } } }' },
  { slug: "supabase", name: "Supabase MCP", category: "Cloud / DB", description: "Manage Supabase DB/projects.", repoUrl: "https://github.com/supabase-community/supabase-mcp", install: { type: "npm", uri: "supabase-mcp" }, configExample: '{ "mcpServers": { "supabase": { "command":"npx","args":["-y","supabase-mcp"] } } }' },
  { slug: "terraform", name: "Terraform MCP", category: "Infra as Code", description: "Terraform plan/apply via LLM.", repoUrl: "https://github.com/devinschumacher/tfmcp", install: { type: "npm", uri: "tfmcp" }, configExample: '{ "mcpServers": { "terraform": { "command":"npx","args":["-y","tfmcp"] } } }' },
  { slug: "docker", name: "Docker MCP", category: "Containers", description: "Manage containers.", repoUrl: "https://github.com/docker-mcp", install: { type: "npm", uri: "docker-mcp" }, configExample: '{ "mcpServers": { "docker": { "command":"npx","args":["-y","docker-mcp"] } } }' },
  { slug: "github", name: "GitHub MCP", category: "Dev", description: "Manage issues, PRs, repos.", repoUrl: "https://github.com/github/github-mcp-server", remote: { apiEndpoint: "https://api.githubcopilot.com/mcp/", provider: "GitHub", authType: "oauth2" }, install: { type: "npm", uri: "github-mcp" }, configExample: '{ "servers": { "github": { "type": "http", "url": "https://api.githubcopilot.com/mcp/" } } }' },
  { slug: "figma", name: "Figma MCP", category: "Design", description: "Extract assets, metadata.", repoUrl: "https://github.com/modelcontextprotocol/figma-mcp", install: { type: "npm", uri: "figma-mcp" }, configExample: '{ "mcpServers": { "figma": { "command":"npx","args":["-y","figma-mcp"] } } }' },
  { slug: "slack", name: "Slack MCP", category: "Comms", description: "Read/post/search Slack.", repoUrl: "https://github.com/augmentcode/slack-mcp-server", install: { type: "npm", uri: "slack-mcp-server" }, configExample: '{ "mcpServers": { "slack": { "command":"npx","args":["-y","slack-mcp-server"] } } }' },
  { slug: "gmail", name: "Gmail MCP", category: "Email", description: "Gmail read/write.", repoUrl: "https://github.com/GongRzhe/Gmail-MCP-Server", install: { type: "npm", uri: "gmail-mcp-server" }, configExample: '{ "mcpServers": { "gmail": { "command":"npx","args":["-y","gmail-mcp-server"] } } }' },
  { slug: "gdocs", name: "Google Docs MCP", category: "Docs", description: "Query/edit Docs.", repoUrl: "https://github.com/modelcontextprotocol/google-docs-mcp", install: { type: "git", uri: "https://github.com/modelcontextprotocol/google-docs-mcp" }, configExample: '{ "mcpServers": { "gdocs": { "command":"node","args":["server.js"], "env":{ "GOOGLE_APPLICATION_CREDENTIALS":"./credentials.json" } } } }' },
  { slug: "gcal", name: "Google Calendar MCP", category: "Calendar", description: "Manage events.", repoUrl: "https://github.com/modelcontextprotocol/google-calendar-mcp", install: { type: "git", uri: "https://github.com/modelcontextprotocol/google-calendar-mcp" }, configExample: '{ "mcpServers": { "gcal": { "command":"node","args":["index.js"], "env":{ "GOOGLE_CLIENT_ID":"...","GOOGLE_CLIENT_SECRET":"..." } } } }' },
  { slug: "notion", name: "Notion MCP", category: "Notes", description: "Manage Notion DB/pages.", repoUrl: "https://github.com/makenotion/notion-mcp-server", remote: { apiEndpoint: "https://mcp.notion.com/mcp", provider: "Notion", authType: "oauth2" }, install: { type: "npm", uri: "@notionhq/notion-mcp-server" }, configExample: '{ "mcpServers": { "Notion": { "url": "https://mcp.notion.com/mcp" } } }' },
  { slug: "youtube", name: "YouTube MCP", category: "Social", description: "Query/fetch YouTube metadata.", repoUrl: "https://github.com/modelcontextprotocol/youtube-mcp", install: { type: "git", uri: "https://github.com/modelcontextprotocol/youtube-mcp" }, configExample: '{ "mcpServers": { "youtube": { "command":"node","args":["server.js"] } } }' },
  { slug: "twitter", name: "TweetBinder MCP", category: "Social", description: "Twitter analytics.", repoUrl: "https://github.com/modelcontextprotocol/tweetbinder-mcp", install: { type: "npm", uri: "tweetbinder-mcp" }, configExample: '{ "mcpServers": { "twitter": { "command":"npx","args":["-y","tweetbinder-mcp"] } } }' },
  { slug: "telegram", name: "Telegram MCP", category: "Messaging", description: "Messaging integration.", repoUrl: "https://github.com/modelcontextprotocol/telegram-mcp", install: { type: "npm", uri: "telegram-mcp" }, configExample: '{ "mcpServers": { "telegram": { "command":"npx","args":["-y","telegram-mcp"] } } }' },
  { slug: "substack", name: "Substack MCP", category: "Social / Newsletters", description: "Manage Substack posts.", repoUrl: "https://github.com/modelcontextprotocol/substack-mcp", install: { type: "npm", uri: "substack-mcp" }, configExample: '{ "mcpServers": { "substack": { "command":"npx","args":["-y","substack-mcp"] } } }' },
  { slug: "prometheus", name: "Prometheus MCP", category: "Monitoring", description: "Query Prometheus metrics.", repoUrl: "https://github.com/modelcontextprotocol/prometheus-mcp", install: { type: "npm", uri: "prometheus-mcp" }, configExample: '{ "mcpServers": { "prometheus": { "command":"npx","args":["-y","prometheus-mcp"] } } }' },
  { slug: "shodan", name: "Shodan MCP", category: "Security / Recon", description: "Shodan scans & OSINT.", repoUrl: "https://github.com/modelcontextprotocol/shodan-mcp", install: { type: "npm", uri: "shodan-mcp" }, configExample: '{ "mcpServers": { "shodan": { "command":"npx","args":["-y","shodan-mcp"] } } }' },
  { slug: "nmap", name: "Nmap MCP", category: "Security / Net", description: "Network scanning.", repoUrl: "https://github.com/modelcontextprotocol/nmap-mcp", install: { type: "npm", uri: "nmap-mcp" }, configExample: '{ "mcpServers": { "nmap": { "command":"npx","args":["-y","nmap-mcp"] } } }' },
];

export const CATEGORIES = Array.from(new Set(CATALOG.map((c) => c.category))).sort();
