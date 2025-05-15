package comments

// 评论系统数据库结构
/*
CREATE TABLE IF NOT EXISTS comments (
    id SERIAL PRIMARY KEY,
    article_id VARCHAR(100) NOT NULL,  -- 对应文章标识符
    nickname VARCHAR(100) NOT NULL,    -- 评论者昵称
    email VARCHAR(100),                -- 评论者邮箱（可选）
    content TEXT NOT NULL,             -- 评论内容
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ip_address VARCHAR(50),            -- 评论者IP地址
    status VARCHAR(20) DEFAULT 'approved',  -- 状态：approved, pending, rejected
    reply_to INTEGER DEFAULT NULL,      -- 回复的评论ID，如果是直接评论则为NULL
    user_agent TEXT                     -- 用户代理信息
)
*/

// 索引
/*
CREATE INDEX IF NOT EXISTS idx_comments_article_id ON comments(article_id);
CREATE INDEX IF NOT EXISTS idx_comments_created_at ON comments(created_at);
CREATE INDEX IF NOT EXISTS idx_comments_status ON comments(status);
CREATE INDEX IF NOT EXISTS idx_comments_reply_to ON comments(reply_to);
*/
