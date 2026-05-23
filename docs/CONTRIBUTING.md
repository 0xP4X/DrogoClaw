# Contributing to DrogonClaw

Thank you for your interest in contributing! This document provides guidelines for contributing to DrogonClaw.

## Code of Conduct

- Be respectful and inclusive
- Report security vulnerabilities privately
- Follow the project's coding standards
- Test your changes thoroughly

## Getting Started

### 1. Fork the Repository

```bash
gh repo fork yourusername/drogonclaw --clone
cd drogonclaw
```

### 2. Create a Feature Branch

```bash
git checkout -b feature/my-feature
```

Use descriptive branch names:
- `feature/add-openai-support`
- `fix/null-pointer-exception`
- `docs/improve-setup-guide`

### 3. Install Dependencies

```bash
npm install
npm run build
npm run lint
```

### 4. Make Changes

Follow the guidelines in `DEVELOPMENT.md`.

Key points:
- Use TypeScript with strict mode
- Add error handling for all async operations
- Use logger instead of console.log
- Write tests for new features
- Update documentation

### 5. Test Your Changes

```bash
npm test
npm run lint
npm run format
```

### 6. Commit Changes

```bash
git commit -m "feature: Add OpenAI support"
```

Follow conventional commits:
- `feature:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation
- `refactor:` - Code refactoring
- `test:` - Adding tests
- `chore:` - Maintenance

### 7. Push and Create PR

```bash
git push origin feature/my-feature
```

Then create a pull request on GitHub.

## Types of Contributions

### Bug Reports

Create an issue with:
- Description of the bug
- Steps to reproduce
- Expected vs actual behavior
- Environment (Node version, OS, etc.)

Template:
```markdown
## Bug Description
[Clear description]

## Steps to Reproduce
1. ...
2. ...

## Expected Behavior
[What should happen]

## Actual Behavior
[What actually happened]

## Environment
- Node: [version]
- OS: [Windows/Mac/Linux]
- DrogonClaw version: [version]
```

### Feature Requests

Propose new features with:
- Use case and motivation
- Proposed implementation
- Examples
- Potential drawbacks

Template:
```markdown
## Feature: [Title]
[Description]

## Motivation
Why this feature is needed

## Implementation Ideas
How to implement it

## Example Usage
```

### Documentation

Improvements to documentation are always welcome:
- Fix typos and grammar
- Improve clarity
- Add examples
- Update outdated information

### Code Improvements

Help with:
- Performance optimization
- Refactoring
- Test coverage
- Error handling

### New Skills

Create new YAML skill definitions:
- Follow the format in `SKILLS.md`
- Test thoroughly
- Document instructions clearly
- Add appropriate remediation advice

Example:
```bash
# Create skill file
vim skills/my-skill.yaml

# Test it
npm start
# Select your skill during pentesting

# Submit PR
git add skills/my-skill.yaml
git commit -m "skill: Add my-skill"
```

## Pull Request Process

### Before Submitting

- [ ] Code follows style guidelines (`npm run lint`)
- [ ] Tests pass (`npm test`)
- [ ] Documentation is updated
- [ ] No console.log statements (use logger)
- [ ] No secrets or API keys in code
- [ ] TypeScript builds without errors (`npm run build`)

### PR Description

Include:
- What problem does this solve?
- How does it solve it?
- Any breaking changes?
- Related issues/PRs

Template:
```markdown
## Description
[Clear description of changes]

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Documentation update
- [ ] Refactoring

## Related Issues
Closes #[issue number]

## Testing
[How to test this change]

## Screenshots
[If applicable]

## Checklist
- [ ] Tests pass
- [ ] Lint passes
- [ ] Documentation updated
- [ ] No breaking changes
```

### Review Process

1. Maintainers review your PR
2. Changes may be requested
3. Update your PR with changes
4. Once approved, it will be merged

## Development Guidelines

### Code Style

```typescript
// ✅ Good
import { logger } from '@/utils/logger.js';

export async function processSession(id: string): Promise<void> {
  try {
    logger.info({ sessionId: id }, 'Processing session');
    const session = await loadSession(id);
    if (!session) {
      throw new Error('Session not found');
    }
    await saveSession(session);
  } catch (err) {
    logger.error({ sessionId: id, error: err }, 'Failed to process session');
    throw err;
  }
}

// ❌ Avoid
console.log('Processing session'); // Use logger
const processSession = async (id) => { // Use function keyword or arrow consistently
  return await db.load(id); // Unnecessary await
};
```

### Error Handling

```typescript
// ✅ Good
try {
  const result = await executeCommand(cmd);
  return result;
} catch (err) {
  logger.error({ command: cmd, error: err }, 'Command failed');
  throw new ToolError(`Failed to execute ${cmd}`, { originalError: err });
}

// ❌ Avoid
const result = await executeCommand(cmd); // No error handling
```

### Testing

```typescript
// ✅ Good
describe('Session Manager', () => {
  it('should load session from database', async () => {
    const session = await storage.loadSession('test-id');
    expect(session).toBeDefined();
    expect(session.id).toBe('test-id');
  });

  it('should throw error if session not found', async () => {
    await expect(storage.loadSession('invalid')).rejects.toThrow();
  });
});

// ❌ Avoid
test('test', () => { // Vague test description
  storage.loadSession(); // No assertions
});
```

### Documentation

```typescript
// ✅ Good
/**
 * Execute a security tool with the given arguments
 * @param toolName - Name of the tool to execute
 * @param args - Command line arguments
 * @returns Tool execution result
 * @throws ToolError if tool execution fails
 */
export async function executeTool(
  toolName: string,
  args: string[],
): Promise<ToolExecution> {
  // implementation
}

// ❌ Avoid
export async function executeTool(toolName, args) { // No JSDoc
  // what does this do?
}
```

## Security

### Reporting Security Vulnerabilities

**DO NOT** create public issues for security vulnerabilities.

Instead:
1. Email: security@drogonclaw.dev
2. Include vulnerability details
3. Allow 90 days for fixes before disclosure

### Security Guidelines

- Never commit API keys or secrets
- Use environment variables for sensitive data
- Validate and sanitize all inputs
- Use parameterized queries for database
- Keep dependencies up to date
- Run security audits: `npm audit`

## Commits

### Commit Message Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

Example:
```
feat(agent): add Claude 3 support

Implements support for Anthropic Claude 3 model with extended thinking.
Adds configuration validation and proper error handling.

Closes #123
```

### Commit Types

- **feat** - New feature
- **fix** - Bug fix
- **docs** - Documentation only
- **style** - Formatting (lint, etc)
- **refactor** - Code refactoring
- **test** - Adding or updating tests
- **chore** - Build, dependencies, etc

### Commit Scopes

- agent
- gateway
- storage
- skills
- channels
- types
- utils
- config
- cli

## Release Process

Maintainers will:
1. Update CHANGELOG.md
2. Update version in package.json
3. Create git tag
4. Push to npm registry
5. Create GitHub release

## Questions?

- Open a discussion: GitHub Discussions
- Check FAQ: docs/FAQ.md
- Email: support@drogonclaw.dev

## Thank You

Thank you for contributing to DrogonClaw! Your efforts help make pentesting more accessible and automated.

---

**Happy Contributing! 🐉**
