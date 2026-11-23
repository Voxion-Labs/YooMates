-- ============================================================================
-- YooMates Production Schema (PostgreSQL 16)
-- Converted from Prisma ORM to Raw High-Performance SQL
-- ============================================================================

-- 0. Extensions & Helper Functions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Enums
DO $$ BEGIN
    CREATE TYPE member_role AS ENUM ('ADMIN', 'MODERATOR', 'MEMBER');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE channel_type AS ENUM ('TEXT', 'AUDIO', 'VIDEO');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE workspace_view_type AS ENUM ('LIST', 'TABLE', 'KANBAN', 'CALENDAR');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- 1. Users & Authentication
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name VARCHAR(255),
    username VARCHAR(64) UNIQUE,
    email VARCHAR(255) UNIQUE,
    email_verified TIMESTAMPTZ,
    image TEXT,
    password_hash TEXT,
    bio TEXT,
    region VARCHAR(64),
    social_links TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS accounts (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(64) NOT NULL,
    provider VARCHAR(64) NOT NULL,
    provider_account_id VARCHAR(255) NOT NULL,
    refresh_token TEXT,
    access_token TEXT,
    expires_at INT,
    token_type VARCHAR(64),
    scope TEXT,
    id_token TEXT,
    session_state TEXT,
    CONSTRAINT uq_accounts_provider_account UNIQUE (provider, provider_account_id)
);
CREATE INDEX IF NOT EXISTS idx_accounts_user_id ON accounts(user_id);

CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    session_token VARCHAR(255) UNIQUE NOT NULL,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);

CREATE TABLE IF NOT EXISTS verification_tokens (
    identifier VARCHAR(255) NOT NULL,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (identifier, token)
);

-- 2. Hubs (Discord Core Collaboration)
CREATE TABLE IF NOT EXISTS servers (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name VARCHAR(255) NOT NULL,
    image_url TEXT,
    invite_code VARCHAR(64) UNIQUE NOT NULL,
    owner_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_servers_owner_id ON servers(owner_id);

CREATE TABLE IF NOT EXISTS members (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    role member_role NOT NULL DEFAULT 'MEMBER',
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    server_id VARCHAR(64) NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_members_user_server UNIQUE (user_id, server_id)
);
CREATE INDEX IF NOT EXISTS idx_members_user_id ON members(user_id);
CREATE INDEX IF NOT EXISTS idx_members_server_id ON members(server_id);

CREATE TABLE IF NOT EXISTS channels (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name VARCHAR(128) NOT NULL,
    type channel_type NOT NULL DEFAULT 'TEXT',
    description TEXT,
    position INT NOT NULL DEFAULT 0,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    server_id VARCHAR(64) NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_channels_server_name UNIQUE (server_id, name)
);
CREATE INDEX IF NOT EXISTS idx_channels_server_id ON channels(server_id);
CREATE INDEX IF NOT EXISTS idx_channels_user_id ON channels(user_id);

CREATE TABLE IF NOT EXISTS direct_message_conversations (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    server_id VARCHAR(64) NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    member_one_id VARCHAR(64) NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    member_two_id VARCHAR(64) NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    user_one_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_two_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_dm_conversation UNIQUE (server_id, member_one_id, member_two_id)
);
CREATE INDEX IF NOT EXISTS idx_dm_server_id ON direct_message_conversations(server_id);
CREATE INDEX IF NOT EXISTS idx_dm_member_one ON direct_message_conversations(member_one_id);
CREATE INDEX IF NOT EXISTS idx_dm_member_two ON direct_message_conversations(member_two_id);

-- Messages Table (Pre-defined without thread_id foreign key to prevent circular reference)
CREATE TABLE IF NOT EXISTS messages (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    content TEXT NOT NULL,
    file_url TEXT,
    file_name VARCHAR(255),
    file_type VARCHAR(64),
    file_size INT,
    member_id VARCHAR(64) NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id VARCHAR(64) REFERENCES channels(id) ON DELETE CASCADE,
    direct_message_conversation_id VARCHAR(64) REFERENCES direct_message_conversations(id) ON DELETE CASCADE,
    is_edited BOOLEAN NOT NULL DEFAULT FALSE,
    forwarded_from_id VARCHAR(64) REFERENCES messages(id) ON DELETE SET NULL,
    thread_id VARCHAR(64),
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_messages_channel_id ON messages(channel_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_messages_dm_id ON messages(direct_message_conversation_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_messages_member_id ON messages(member_id);
CREATE INDEX IF NOT EXISTS idx_messages_thread_id ON messages(thread_id);

CREATE TABLE IF NOT EXISTS threads (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    title VARCHAR(255),
    server_id VARCHAR(64) NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    channel_id VARCHAR(64) REFERENCES channels(id) ON DELETE CASCADE,
    direct_message_conversation_id VARCHAR(64) REFERENCES direct_message_conversations(id) ON DELETE CASCADE,
    root_message_id VARCHAR(64) UNIQUE NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    creator_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_threads_server_id ON threads(server_id);
CREATE INDEX IF NOT EXISTS idx_threads_channel_id ON threads(channel_id);
CREATE INDEX IF NOT EXISTS idx_threads_creator_id ON threads(creator_id);

-- Attach messages.thread_id constraint
DO $$ BEGIN
    ALTER TABLE messages ADD CONSTRAINT fk_messages_thread 
    FOREIGN KEY (thread_id) REFERENCES threads(id) ON DELETE CASCADE;
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS reactions (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    emoji VARCHAR(32) NOT NULL,
    message_id VARCHAR(64) NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    member_id VARCHAR(64) NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_reactions_msg_member_emoji UNIQUE (message_id, member_id, emoji)
);
CREATE INDEX IF NOT EXISTS idx_reactions_message_id ON reactions(message_id);

-- 3. Global Feed (Twitter Core)
CREATE TABLE IF NOT EXISTS posts (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    content TEXT NOT NULL,
    media_url TEXT,
    media_type VARCHAR(64),
    media_name VARCHAR(255),
    author_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    quote_post_id VARCHAR(64) REFERENCES posts(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_posts_author_id ON posts(author_id);
CREATE INDEX IF NOT EXISTS idx_posts_quote_post_id ON posts(quote_post_id);
CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC);

CREATE TABLE IF NOT EXISTS comments (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    text TEXT NOT NULL,
    post_id VARCHAR(64) NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    author_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_comments_post_id ON comments(post_id, created_at ASC);

CREATE TABLE IF NOT EXISTS likes (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    post_id VARCHAR(64) NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_likes_post_user UNIQUE (post_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_likes_user_id ON likes(user_id);

CREATE TABLE IF NOT EXISTS bookmarks (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    post_id VARCHAR(64) NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_bookmarks_post_user UNIQUE (post_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_bookmarks_user_id ON bookmarks(user_id);

CREATE TABLE IF NOT EXISTS polls (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    post_id VARCHAR(64) UNIQUE NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    question TEXT,
    allow_multiple_votes BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS poll_options (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    poll_id VARCHAR(64) NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    position INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_poll_options_position UNIQUE (poll_id, position)
);

CREATE TABLE IF NOT EXISTS poll_votes (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    poll_id VARCHAR(64) NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    option_id VARCHAR(64) NOT NULL REFERENCES poll_options(id) ON DELETE CASCADE,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_poll_votes_option_user UNIQUE (option_id, user_id)
);

-- 4. My Space (Notion Core)
CREATE TABLE IF NOT EXISTS document_folders (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name VARCHAR(255) NOT NULL,
    color VARCHAR(32),
    position INT NOT NULL DEFAULT 0,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_doc_folders_user_id ON document_folders(user_id);

CREATE TABLE IF NOT EXISTS documents (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    icon VARCHAR(64),
    cover_image_url TEXT,
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    is_published BOOLEAN NOT NULL DEFAULT FALSE,
    parent_id VARCHAR(64) REFERENCES documents(id) ON DELETE SET NULL,
    position INT NOT NULL DEFAULT 0,
    status VARCHAR(64),
    due_date TIMESTAMPTZ,
    metadata JSONB,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    folder_id VARCHAR(64) REFERENCES document_folders(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_documents_user_id ON documents(user_id);
CREATE INDEX IF NOT EXISTS idx_documents_folder_id ON documents(folder_id);
CREATE INDEX IF NOT EXISTS idx_documents_parent_id ON documents(parent_id);

CREATE TABLE IF NOT EXISTS document_reminders (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    title VARCHAR(255) NOT NULL,
    note TEXT,
    remind_at TIMESTAMPTZ NOT NULL,
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    document_id VARCHAR(64) NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_doc_reminders_remind_at ON document_reminders(remind_at);

CREATE TABLE IF NOT EXISTS workspace_views (
    id VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name VARCHAR(255) NOT NULL,
    type workspace_view_type NOT NULL DEFAULT 'LIST',
    position INT NOT NULL DEFAULT 0,
    config JSONB,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    folder_id VARCHAR(64) NOT NULL REFERENCES document_folders(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_workspace_views_name UNIQUE (folder_id, name),
    CONSTRAINT uq_workspace_views_pos UNIQUE (folder_id, position)
);
CREATE INDEX IF NOT EXISTS idx_workspace_views_folder ON workspace_views(folder_id);
