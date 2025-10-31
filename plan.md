## UI 实现与设计稿差异
- `FlowLayout` 侧边栏额外加入 “AI” 方形 logo 与圆角步骤按钮，未复刻设计稿里的纯文字品牌与带竖分隔的 Step 链接样式（frontend/src/components/flow/FlowLayout.vue:4-58，对比 ui/step 1/code.html）。
- 首页仍采用 `FlowLayout` 的标题+卡片排版，且输入框 placeholder 改为网址示例，未呈现设计稿中居中布局与“输入客户公司全名或官网地址”提示（frontend/src/pages/HomePage.vue:2-17，对比 ui/首页/code.html）。
- Step 1 页面新增了公司名称、AI 摘要、候选官网等表单和“重新获取 AI 信息”按钮，联系人区域也多出“来源”和“重点”控件，底部按钮改成“返回上一步 / 保存并继续”，均与设计稿不符（frontend/src/components/flow/StepOne.vue:14-100，对比 ui/step 1/code.html）。
- Step 2 需要手动点击 “获取/重新评估” 才触发评级，并出现额外的底部导航按钮，设计稿则展示自动评估流程与单个“下一步”操作（frontend/src/components/flow/StepTwo.vue:35-58，对比 ui/step2/code.html）。
- Step 3 顶部提供的是“生成/重新生成”入口且内容默认可编辑，设计稿期望的是先展示只读结果并通过“编辑”按钮切换状态（frontend/src/components/flow/StepThree.vue:12-60，对比 ui/step3/code.html）。
- Step 4 需先点击“生成开发信”按钮才能出现正文，并保持 “返回上一步 / 保存并继续” 按钮，而设计稿默认展示邮件草稿并提供“保存为首次跟进记录”单按钮操作（frontend/src/components/flow/StepFour.vue:18-56，对比 ui/step4/code.html）。
- Step 5 增加了加载态、Material 图标和“返回上一步”按钮，成功提示改为显示具体时间戳，未与设计稿的纯文案提示保持一致（frontend/src/components/flow/StepFive.vue:12-40，对比 ui/step5/code.html）。
- 客户列表新增搜索框、状态列及更多排序方式，与设计稿只包含基础筛选和“最新跟进日期”列的表格不一致（frontend/src/pages/CustomersPage.vue:4-83，对比 ui/客户管理页面/code.html）。
- 客户编辑弹窗增加公司名称、摘要、“来源/重点”字段以及 `UNKNOWN` 选项，国家选择变成纯文本输入，也与设计稿的字段和选项集不同（frontend/src/components/customers/CustomerEditModal.vue:18-78，对比 ui/客户信息编辑页面/code.html）。
- 全局配置页仍使用仅有三项的导航与单个“保存配置”按钮，缺少设计稿中的扩展菜单与底部“取消/保存更改”双按钮（frontend/src/pages/SettingsPage.vue:2-147，对比 ui/全局配置页面_1/code.html）。

## 前端实现与后端接口结构差异
- 客户编辑弹窗更新公司信息时把 `source_json` 强制写成空对象，覆盖掉后端用于回溯 AI 聚合结果的原始数据（frontend/src/components/customers/CustomerEditModal.vue:309-316，后端读取在 backend/store/customers.go:320-340）。  
- 若客户从未生成过分析报告，弹窗保存时不会调用 `UpdateAnalysis`，导致新填写的分析内容无法写入后端（frontend/src/components/customers/CustomerEditModal.vue:322-329，接口在 backend/api/handlers.go:328-340）。

## 前端页面 vs 数据库存储/存储过程差异
- 联系人表单未提供电话字段，无法维护 `contacts.phone` 列（frontend/src/components/flow/StepOne.vue:68-91、frontend/src/components/customers/CustomerEditModal.vue:47-58，对比 backend/store/store.go:78-88）。
- 首次跟进流程没有采集备注，前端始终以空字符串调用 `SaveInitialFollowup`，使得 `followups.notes` 列形同虚设（frontend/src/stores/flow.js:233-245，对比 backend/store/store.go:115-124）。
- 调度结果只在 UI 中展示计划时间，数据库还保存 `scheduled_tasks.status/last_error` 等字段，但界面无处查看或校验（frontend/src/components/flow/StepFive.vue:32-35，对比 backend/store/store.go:125-138）。

## 页面切换与按钮交互问题
- Step 2 未进入页面就自动提交评级，用户必须额外点击“获取 AI 评级”，与设计稿即时评估的体验不符，也会让流程停留在第二步（frontend/src/components/flow/StepTwo.vue:35-44）。
- Step 4 在用户生成邮件之前 “保存并继续” 被禁用，设计稿期望的是直接查看并保存草稿而非额外一步生成动作（frontend/src/components/flow/StepFour.vue:24-56）。
- Step 1 的“保存并继续”要求公司名称非空，但设计稿只提供官网/国家字段，可能导致按设计操作时无法继续流程（frontend/src/components/flow/StepOne.vue:96-100）。
