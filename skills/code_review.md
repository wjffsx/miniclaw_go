---
name: "code_review"
description: "Review code for best practices, security issues, and performance optimizations"
category: "development"
tags: ["code", "review", "quality", "best-practices"]
requires: ["read_file"]
---

# Code Review Skill

This skill helps you review code for best practices, security issues, and performance optimizations.

## When to Use

Use this skill when:
- User asks for code review
- User mentions "review" or "check my code"
- User shares code snippets
- User asks about code quality

## How to Use

1. Read the code file using `read_file`
2. Analyze the code for:
   - Security vulnerabilities (SQL injection, XSS, etc.)
   - Performance issues (inefficient algorithms, memory leaks)
   - Code style violations (naming conventions, formatting)
   - Best practices adherence (error handling, logging)
3. Provide actionable feedback with specific examples
4. Suggest improvements with code examples

## Review Checklist

### Security
- Input validation and sanitization
- SQL injection prevention
- XSS prevention
- Authentication and authorization
- Sensitive data handling

### Performance
- Algorithm efficiency
- Memory usage
- Database query optimization
- Caching strategies
- Concurrency and parallelism

### Code Quality
- Naming conventions
- Code organization
- Error handling
- Logging and monitoring
- Testing coverage

### Best Practices
- Design patterns
- SOLID principles
- DRY (Don't Repeat Yourself)
- KISS (Keep It Simple, Stupid)
- Documentation

## Example Response

When reviewing code, structure your response as follows:

1. **Summary**: Brief overview of the code
2. **Security Issues**: List any security concerns
3. **Performance Issues**: List any performance concerns
4. **Code Quality Issues**: List any code quality concerns
5. **Suggestions**: Provide specific improvements with code examples
6. **Overall Rating**: Give a score (1-10) and brief explanation
