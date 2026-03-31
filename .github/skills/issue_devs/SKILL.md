name: issue_devs
description: GitHub Issueの内容をもとに実装方針を自動生成し、ユーザーの確認後に実装へ進むための支援を行うSkill

# Issue Implementation Planner Skill

## 概要

このSkillは、GitHub Issueの内容をもとに実装方針を自動生成し、ユーザーの確認後に実装へ進むための支援を行います。

主な目的は以下です：

* Issueの内容を構造化して理解する
* 実装可能なレベルの具体的な計画を生成する
* 計画をMarkdownとして保存する
* ユーザーの合意を得たうえで実装に進む

---

## 入力

以下の情報を受け取ります：

* Issue番号

---

## 出力

以下の形式でMarkdownを生成します：

```md
# Issue #{issue_number}: {issue_title}

## Description
Issueの内容を要約して記述

## Summary
概要（2〜3文）

## Requirements
- 機能要件
- 非機能要件

## Implementation Approach
実装方針（設計・技術選定・データフローなど）

## Task Breakdown
1. タスク1
2. タスク2

## Affected Files / Components
変更・追加されるファイルやモジュール

## Edge Cases & Risks
- 注意点
- バグになり得るケース
- セキュリティやパフォーマンス上の懸念

## Open Questions
不明点・確認事項
```

---

## 処理フロー

1. GitHub Issueを取得
    gh auth login --with-tokenを実行し、未ログイン状態であれば、ユーザーにGitHubログインを促す
2. Issue内容を解析
3. 実装計画を生成
4. 以下のパスに保存：
    ```
    docs/issues/#{issue_number}_{issue_title}.md
    ```
5. 計画内容をユーザーに表示
6. ユーザーに確認：

* Yes → 実装フェーズへ進む
* No → 修正または中断

---

## 振る舞いルール

* 抽象的な説明ではなく、**実装可能な具体性**を重視する
* タスクはそのまま作業に移せる粒度で分解する
* 既存コードベースは一般的なベストプラクティスに従っている前提とする
* 不明点は必ず「Open Questions」として明示する
* 推測で実装を決め打ちしない

---

## 実装フェーズ（確認後）

ユーザーが承認した場合：

* 必要なファイルを作成・編集
* コードを生成
* 既存コードに統合
* ブランチ作成

---

## 想定コマンド

```bash
issue_devs <issue_number>
```

## ブランチ
ブランチが存在しない場合は作成する
命名規則: `feature/issue-{issue_number}`

---

## 備考

* `--dry-run` オプションで計画生成のみ実行可能
* `--auto` オプションで確認をスキップ可能
* 将来的にPR自動作成との連携を想定
