'use client';

import { useState } from 'react';
import '../styles/DMPage.css';

export default function DMPage({ onNavigate, isDarkMode, toggleTheme }) {
  const [selectedChat, setSelectedChat] = useState('alice');
  const [messageText, setMessageText] = useState('');
  const [conversations, setConversations] = useState({
    alice: {
      name: 'Alice',
      avatar: '👩',
      isRemote: false,
      server: 'social.example.net',
      messages: [
        {
          id: 1,
          sender: 'alice',
          content: 'Hey! How are you?',
          timestamp: '10:30 AM',
          isSent: false,
          isEncrypted: true,
        },
        {
          id: 2,
          sender: 'you',
          content: "I'm good! Just working on federation stuff.",
          timestamp: '10:31 AM',
          isSent: true,
          isEncrypted: true,
        },
        {
          id: 3,
          sender: 'alice',
          content: 'Nice! Want to collaborate on the DID system?',
          timestamp: '10:32 AM',
          isSent: false,
          isEncrypted: true,
        },
      ],
    },
    bob: {
      name: 'Bob',
      avatar: '👨',
      isRemote: true,
      server: 'federated.social',
      messages: [
        {
          id: 1,
          sender: 'bob',
          content: 'Your instance federation looks solid 🚀',
          timestamp: '9:15 AM',
          isSent: false,
          isEncrypted: true,
        },
        {
          id: 2,
          sender: 'you',
          content: 'Thanks! E2E encryption with your server working great.',
          timestamp: '9:16 AM',
          isSent: true,
          isEncrypted: true,
        },
      ],
    },
    carol: {
      name: 'Carol',
      avatar: '👩‍💼',
      isRemote: false,
      server: 'social.example.net',
      messages: [
        {
          id: 1,
          sender: 'carol',
          content: 'Meeting at 2pm?',
          timestamp: '8:00 AM',
          isSent: false,
          isEncrypted: true,
        },
      ],
    },
  });

  const handleSendMessage = () => {
    if (messageText.trim()) {
      const updatedConversations = { ...conversations };
      updatedConversations[selectedChat].messages.push({
        id:
          updatedConversations[selectedChat].messages.length +
          1,
        sender: 'you',
        content: messageText,
        timestamp: new Date().toLocaleTimeString([], {
          hour: '2-digit',
          minute: '2-digit',
        }),
        isSent: true,
        isEncrypted: true,
      });
      setConversations(updatedConversations);
      setMessageText('');
    }
  };

  const currentChat = conversations[selectedChat];

  return (
    <div className="dm-container">
      {/* Top Navigation */}
      <div className="dm-navbar">
        <button
          className="nav-button back-button"
          onClick={() => onNavigate('home')}
        >
          ← Back
        </button>
        <h1 className="navbar-title">Messages 🔒</h1>
        <div style={{ marginLeft: 'auto', display: 'flex', gap: '10px' }}>
          <button onClick={() => onNavigate('profile')} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>👤 Profile</button>
          <button onClick={() => onNavigate('security')} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>🔐 Security</button>
          <button onClick={toggleTheme} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>{isDarkMode ? '🌙' : '☀️'}</button>
        </div>
      </div>

      <div className="dm-content">
        {/* Sidebar - Inbox */}
        <div className="dm-sidebar">
          <div className="sidebar-header">
            <h2>Inbox</h2>
          </div>
          <div className="conversations-list">
            {Object.entries(conversations).map(([key, chat]) => (
              <div
                key={key}
                className={`conversation-item ${selectedChat === key ? 'active' : ''}`}
                onClick={() => setSelectedChat(key)}
              >
                <div className="conversation-avatar">{chat.avatar}</div>
                <div className="conversation-info">
                  <div className="conversation-name">
                    @{chat.name.toLowerCase()}
                    {chat.isRemote && (
                      <span className="remote-indicator">🌐</span>
                    )}
                  </div>
                  <div className="conversation-preview">
                    {chat.messages[chat.messages.length - 1].content.substring(
                      0,
                      30
                    )}
                    ...
                  </div>
                </div>
                <div className="encryption-badge">🔒</div>
              </div>
            ))}
          </div>
        </div>

        {/* Chat Window */}
        <div className="dm-chat-window">
          {/* Chat Header */}
          <div className="chat-header">
            <div className="chat-header-info">
              <div className="chat-avatar">{currentChat.avatar}</div>
              <div className="chat-title-section">
                <h2 className="chat-title">@{currentChat.name.toLowerCase()}</h2>
                <div className="chat-status">
                  {currentChat.isRemote && (
                    <span className="status-text">🌐 Remote • </span>
                  )}
                  <span className="status-text">🔒 Encrypted</span>
                </div>
              </div>
            </div>
          </div>

          {/* Encryption Banner */}
          <div className="encryption-banner">
            🔒 Messages are end-to-end encrypted. The server cannot read this
            content.
          </div>

          {/* Messages Area */}
          <div className="messages-area">
            {currentChat.messages.map((message) => (
              <div
                key={message.id}
                className={`message-bubble ${message.isSent ? 'sent' : 'received'}`}
              >
                <div className="message-content">{message.content}</div>
                <div className="message-meta">
                  <span className="message-timestamp">{message.timestamp}</span>
                  <span className="message-encryption" title="End-to-end encrypted">
                    🔒
                  </span>
                </div>
              </div>
            ))}
          </div>

          {/* Input Box */}
          <div className="message-input-box">
            <textarea
              className="message-input"
              placeholder="Type a message..."
              value={messageText}
              onChange={(e) => setMessageText(e.target.value)}
              onKeyPress={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault();
                  handleSendMessage();
                }
              }}
            ></textarea>
            <button
              className="send-button"
              onClick={handleSendMessage}
              disabled={!messageText.trim()}
            >
              Send 🔒
            </button>
          </div>

          {/* Disabled Features */}
          <div className="disabled-features">
            <button className="feature-button disabled" disabled>
              📎 Attachments
            </button>
            <span className="feature-tooltip">Sprint 2</span>
            <button className="feature-button disabled" disabled>
              🔄 Multi-device Sync
            </button>
            <span className="feature-tooltip">Sprint 3</span>
          </div>
        </div>
      </div>
    </div>
  );
}
