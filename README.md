# gocron

## 功能

实现简单的定时器功能：

1. 时间精度为毫秒
2. 支持周期和非周期性定时任务
3. 支持定时任务O(1)时间复杂度的增删

## 接口

1. 创建定时器

```
func New() *Cron
```

2. 启动定时器
```
func (c *Cron) Start()
```

3. 关闭定时器
```
func (c *Cron) Stop() context.Context 
```

4. 添加定时任务

```
// 入参：
// repeat：任务重复执行次数， -1 表示为周期任务
// spec: 定时任务周期, 格式支持2种：
//           “@every ” 
//           "month,d,h,min,s,ms" (数值用数字或*表示，数字表示确定的时间值，*表示任意时间值)
// f: 定时任务具体操作
//
// 出参：
// 定时任务ID
// error
func (c *Cron) AddJob(repeat int64, spec string, f func()) (int, error)
```

5. 删除定时任务
```
func (c *Cron) RemoveJob(id int)
```
















