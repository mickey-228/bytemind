# Extensions Config

`extensions` 配置块用于控制 extension 运行时的来源、自动加载和故障恢复策略。

示例：

```json
{
  "extensions": {
    "sources": ["skills", "mcp"],
    "auto_load": true,
    "health_check_interval_sec": 30,
    "failure_threshold": 3,
    "recovery_cooldown_sec": 30,
    "max_concurrency_per_extension": 4,
    "conflict_policy": "reject"
  }
}
```

字段说明：

- `sources`：启用的扩展来源，当前支持 `skills`、`mcp`。
- `auto_load`：启动时是否自动发现并加载扩展。
- `health_check_interval_sec`：健康巡检间隔（秒）。
- `failure_threshold`：连续失败达到该阈值后进入熔断。
- `recovery_cooldown_sec`：熔断后再次探测前的冷却时间（秒）。
- `max_concurrency_per_extension`：单扩展最大并发执行数。
- `conflict_policy`：同名冲突策略，支持 `reject`、`first_wins`、`last_wins`。

