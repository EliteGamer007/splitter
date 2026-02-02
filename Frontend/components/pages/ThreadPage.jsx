'use client';

import { useState } from 'react';
import { useTheme } from '@/components/ui/theme-provider';
import '../styles/ThreadPage.css';

export default function ThreadPage({ onNavigate }) {
  const { theme, toggleTheme } = useTheme();
  const isDarkMode = theme === 'dark';
  const [replies, setReplies] = useState([
    {
      id: 1,
      author: 'alice',
      server: 'social.example.net',
      avatar: '👩',
      content: 'Just deployed a new federated instance! 🚀',
      timestamp: '2 hours ago',
      isLocal: true,
      isFederated: false,
      replies: 2,
      boosts: 12,
      likes: 45,
      depth: 0,
    },
    {
      id: 2,
      author: 'bob',
      server: 'federated.social',
      avatar: '👨',
      content: 'That looks amazing! How did you set up federation?',
      timestamp: '1 hour ago',
      isLocal: false,
      isFederated: true,
      replies: 1,
      boosts: 8,
      likes: 22,
      depth: 1,
    },
    {
      id: 3,
      author: 'alice',
      server: 'social.example.net',
      avatar: '👩',
      content: 'Using ActivityPub with WebFinger discovery. Took a while to get signature validation right.',
      timestamp: '45 minutes ago',
      isLocal: true,
      isFederated: false,
      replies: 0,
      boosts: 5,
      likes: 18,
      depth: 2,
    },
    {
      id: 4,
      author: 'carol',
      server: 'privacy.social',
      avatar: '👩‍💼',
      content: 'This is exactly what the decentralized web needs!',
      timestamp: '30 minutes ago',
      isLocal: false,
      isFederated: true,
      replies: 0,
      boosts: 15,
      likes: 56,
      depth: 1,
    },
  ]);

  const [replyText, setReplyText] = useState('');

  const handleReply = () => {
    if (replyText.trim()) {
      const newReply = {
        id: replies.length + 1,
        author: 'you',
        server: 'social.example.net',
        avatar: '🧑',
        content: replyText,
        timestamp: 'now',
        isLocal: true,
        isFederated: false,
        replies: 0,
        boosts: 0,
        likes: 0,
        depth: 1,
      };
      setReplies([...replies, newReply]);
      setReplyText('');
    }
  };

  return (
    <div className="thread-container">
      {/* Top Navigation */}
      <div className="thread-navbar">
        <button
          className="nav-button back-button"
          onClick={() => onNavigate('home')}
        >
          ← Back to Timeline
        </button>
        <div style={{ marginLeft: 'auto', display: 'flex', gap: '10px' }}>
          <button onClick={() => onNavigate('profile')} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>Profile</button>
          <button onClick={() => onNavigate('dm')} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>💬 Messages</button>
          <button onClick={toggleTheme} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>{isDarkMode ? '🌙' : '☀️'}</button>
        </div>
      </div>

      <div className="thread-content">
        {/* Root Post */}
        <div className="root-post-card">
          <div className="post-author-section">
            <div className="post-avatar">👩</div>
            <div className="post-author-info">
              <div className="post-author-name">@alice@social.example.net</div>
              <div className="post-timestamp">2 hours ago</div>
            </div>
            <span className="local-badge">Local</span>
          </div>

          <div className="post-body">
            <p>Just deployed a new federated instance! 🚀</p>
            <p>
              The challenge-response authentication system is working perfectly
              with cross-instance key verification.
            </p>
          </div>

          <div className="post-stats">
            <span>45 Likes</span>
            <span>12 Boosts</span>
            <span>8 Replies</span>
          </div>
        </div>

        {/* Replies Section */}
        <div className="replies-section">
          <div className="thread-line"></div>

          {replies.map((reply) => (
            <div
              key={reply.id}
              className={`reply-card depth-${reply.depth}`}
              style={{ marginLeft: `${reply.depth * 20}px` }}
            >
              {reply.depth > 0 && <div className="thread-connector"></div>}

              <div className="reply-author-section">
                <div className="reply-avatar">{reply.avatar}</div>
                <div className="reply-author-info">
                  <div className="reply-author-name">
                    @{reply.author}@{reply.server}
                  </div>
                  {reply.isFederated && (
                    <span
                      className="federated-indicator"
                      title="Fetched from remote server"
                    >
                      🌐
                    </span>
                  )}
                  <div className="reply-timestamp">{reply.timestamp}</div>
                </div>
              </div>

              <div className="reply-body">{reply.content}</div>

              <div className="reply-actions">
                <button
                  className={`action-button ${reply.isFederated ? 'disabled' : ''}`}
                  title={
                    reply.isFederated
                      ? 'Cross-server interaction federation enabled in Sprint 2'
                      : ''
                  }
                  disabled={reply.isFederated}
                >
                  ♥ {reply.likes}
                </button>
                <button
                  className={`action-button ${reply.isFederated ? 'disabled' : ''}`}
                  disabled={reply.isFederated}
                >
                  🔄 {reply.boosts}
                </button>
                <button
                  className={`action-button ${reply.isFederated ? 'disabled' : ''}`}
                  disabled={reply.isFederated}
                >
                  💬 {reply.replies}
                </button>
              </div>
            </div>
          ))}
        </div>

        {/* Reply Composer */}
        <div className="reply-composer">
          <textarea
            className="reply-textarea"
            placeholder="Write your reply..."
            value={replyText}
            onChange={(e) => setReplyText(e.target.value)}
          ></textarea>
          <div className="composer-actions">
            <span className="char-count">
              {replyText.length} / 500 characters
            </span>
            <button
              className="reply-button"
              onClick={handleReply}
              disabled={!replyText.trim()}
            >
              Reply
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
