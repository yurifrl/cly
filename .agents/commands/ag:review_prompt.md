# Command: /review_prompt

Review and improve AI prompts using Google's Gemini prompting strategies.

## Usage

```
/review_prompt <prompt text or file path>
```

## Behavior

1. **Analyze the prompt** against established best practices:
   - Clear and specific instructions
   - Appropriate use of examples (few-shot vs zero-shot)
   - Context placement and structure
   - Constraint definition
   - Output format specification

2. **Identify issues**:
   - Vague or ambiguous instructions
   - Missing examples where needed
   - Poor context organization
   - Unclear output expectations
   - Buried critical instructions

3. **Provide improvements**:
   - Restructured prompt with clearer sections
   - Added examples if beneficial
   - Better constraint definitions
   - Optimized context placement
   - Enhanced specificity

4. **Explain reasoning** for each suggested change

## Examples

<example>
User: /review_prompt "Write some code for authentication"
Assistant:
**Issues Found:**
- Too vague: "some code" doesn't specify what exactly
- No context: what language, framework, requirements?
- No constraints: security requirements, scope
- No format: what should output look like?

**Improved Prompt:**

```xml
<role>You are an experienced backend developer specializing in secure authentication.</role>

<constraints>
1. Use JWT tokens for stateless authentication
2. Implement password hashing with bcrypt
3. Include rate limiting for login attempts
4. Follow OWASP security guidelines
</constraints>

<context>
Language: Python 3.11
Framework: FastAPI
Database: PostgreSQL with SQLAlchemy ORM
</context>

<task>
Create an authentication module with:
- User registration endpoint
- Login endpoint with JWT generation
- Token validation middleware
- Password reset flow

Output format: Complete Python module with docstrings
</task>
```

**Reasoning:**
- Added specific role for domain expertise
- Defined security constraints explicitly
- Provided technical context (language, framework)
- Broke down task into clear deliverables
- Specified output format
</example>

<example>
User: /review_prompt home/.claude/skills/testing.md
Assistant:
[Reads file and analyzes structure]

**Strengths:**
✅ Clear "When to Use" guidance
✅ Strong anti-pattern warnings (mockist trap)
✅ Decision logic flowchart
✅ Concrete examples

**Potential Improvements:**
- Add 2-3 code examples showing good vs bad tests
- Include explicit output format for test results
- Add section on test naming conventions

**Suggested Addition:**

```markdown
## Example: Good vs Bad Unit Test

### ❌ Bad (over-mocked)
```python
def test_user_service():
    mock_db = Mock()
    mock_auth = Mock()
    mock_logger = Mock()
    service = UserService(mock_db, mock_auth, mock_logger)
    # This tests nothing real
```

### ✅ Good (minimal mocking)
```python
def test_user_service():
    db = InMemoryDatabase()
    service = UserService(db, real_auth, real_logger)
    user = service.create_user("test@example.com")
    assert db.users.contains(user.id)
```
```

**Reasoning:**
- Examples make abstract guidance concrete
- Visual comparison highlights the difference clearly
- Maintains existing structure while enhancing clarity
</example>

## Review Criteria

The command evaluates prompts on:

1. **Clarity** (1-5): Are instructions unambiguous?
2. **Specificity** (1-5): Are requirements concrete?
3. **Structure** (1-5): Is information organized logically?
4. **Examples** (1-5): Are appropriate examples provided?
5. **Constraints** (1-5): Are boundaries clearly defined?
6. **Format** (1-5): Is output format specified?

**Overall Score**: Average of above (1-5 scale)

## Notes

- This command invokes the `prompt-engineering` skill automatically
- Works with both inline prompts and file paths
- For commands/skills/agents, considers Claude Code specific patterns
- Prioritizes actionable feedback over theory
