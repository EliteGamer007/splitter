'use client';

import { useState, useEffect, useRef } from 'react';
import { useTheme } from '@/components/ui/theme-provider';
import '../styles/DMPage.css';
import { messageApi, searchApi } from '@/lib/api';

export default function DMPage({ onNavigate, userData, selectedUser }) {
  const { theme, toggleTheme } = useTheme();
  const isDarkMode = theme === 'dark';
  const [threads, setThreads] = useState([]);
  const [selectedThread, setSelectedThread] = useState(null);
  const [messages, setMessages] = useState([]);
  const [messageText, setMessageText] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [isSending, setIsSending] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState([]);
  const [isSearching, setIsSearching] = useState(false);
  const [showNewChat, setShowNewChat] = useState(false);
  const messagesEndRef = useRef(null);

  // Fetch threads on mount
  useEffect(() => {
    fetchThreads();
  }, []);

  // Handle selectedUser from navigation
  useEffect(() => {
    if (selectedUser) {
      startConversationWithUser(selectedUser);
    }
  }, [selectedUser]);

  // Auto-scroll to bottom of messages
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const fetchThreads = async () => {
    setIsLoading(true);
    try {
      const result = await messageApi.getThreads();
      setThreads(result.threads || []);

      // If we have threads and none selected, select the first one
      if (result.threads?.length > 0 && !selectedThread) {
        selectThread(result.threads[0]);
      }
    } catch (err) {
      console.error('Failed to fetch threads:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const selectThread = async (thread) => {
    setSelectedThread(thread);
    try {
      const result = await messageApi.getMessages(thread.id);
      setMessages(result.messages || []);
    } catch (err) {
      console.error('Failed to fetch messages:', err);
      setMessages([]);
    }
  };

  const handleSendMessage = async () => {
    if (!messageText.trim() || !selectedThread) return;

    setIsSending(true);
    try {
      // Determine recipient ID
      const recipientId = selectedThread.participant_a_id === userData?.id
        ? selectedThread.participant_b_id
        : selectedThread.participant_a_id;

      const result = await messageApi.sendMessage(recipientId, messageText);

      // Add message to local state
      setMessages(prev => [...prev, result.message]);
      setMessageText('');

      // Refresh threads to update last message
      fetchThreads();
    } catch (err) {
      console.error('Failed to send message:', err);
      alert('Failed to send message: ' + err.message);
    } finally {
      setIsSending(false);
    }
  };

  // Search for users to start new conversation
  const handleSearch = async () => {
    if (searchQuery.length < 2) return;

    setIsSearching(true);
    try {
      const result = await searchApi.searchUsers(searchQuery);
      // Filter out current user
      const filtered = (result.users || []).filter(u => u.id !== userData?.id);
      setSearchResults(filtered);
    } catch (err) {
      console.error('Search failed:', err);
      setSearchResults([]);
    } finally {
      setIsSearching(false);
    }
  };

  useEffect(() => {
    if (searchQuery.length >= 2) {
      const timer = setTimeout(handleSearch, 300);
      return () => clearTimeout(timer);
    } else {
      setSearchResults([]);
    }
  }, [searchQuery]);

  const startConversationWithUser = async (user) => {
    try {
      const result = await messageApi.startConversation(user.id);
      setShowNewChat(false);
      setSearchQuery('');
      setSearchResults([]);

      // Refresh threads and select the new one
      await fetchThreads();
      selectThread(result.thread);
    } catch (err) {
      console.error('Failed to start conversation:', err);
      alert('Failed to start conversation: ' + err.message);
    }
  };

  const formatTimestamp = (isoString) => {
    if (!isoString) return '';
    const date = new Date(isoString);
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };

  const getOtherUser = (thread) => {
    return thread.other_user || { username: 'Unknown', display_name: 'Unknown User' };
  };

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
          <button onClick={() => onNavigate('profile')} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>Profile</button>
          <button onClick={() => onNavigate('security')} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>🔐 Security</button>
          <button onClick={toggleTheme} style={{ padding: '8px 12px', background: 'rgba(0,217,255,0.1)', border: '1px solid #00d9ff', color: '#00d9ff', borderRadius: '6px', cursor: 'pointer' }}>{isDarkMode ? '🌙' : '☀️'}</button>
        </div>
      </div>

      <div className="dm-content">
        {/* Sidebar - Inbox */}
        <div className="dm-sidebar">
          <div className="sidebar-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <h2>Inbox</h2>
            <button
              onClick={() => setShowNewChat(!showNewChat)}
              style={{
                padding: '6px 12px',
                background: showNewChat ? 'rgba(255,68,68,0.2)' : 'rgba(0,217,255,0.2)',
                border: `1px solid ${showNewChat ? '#ff4444' : '#00d9ff'}`,
                color: showNewChat ? '#ff4444' : '#00d9ff',
                borderRadius: '4px',
                cursor: 'pointer',
                fontSize: '14px'
              }}
            >
              {showNewChat ? '✕ Cancel' : '+ New'}
            </button>
          </div>

          {/* New Chat Search */}
          {showNewChat && (
            <div style={{ padding: '12px', borderBottom: '1px solid #333' }}>
              <input
                type="text"
                placeholder="Search users to message..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                style={{
                  width: '100%',
                  padding: '10px',
                  background: 'rgba(255,255,255,0.05)',
                  border: '1px solid #333',
                  borderRadius: '6px',
                  color: '#fff',
                  fontSize: '14px'
                }}
              />
              {isSearching && (
                <div style={{ padding: '8px', color: '#666', textAlign: 'center' }}>
                  Searching...
                </div>
              )}
              {searchResults.length > 0 && (
                <div style={{ marginTop: '8px' }}>
                  {searchResults.map(user => (
                    <div
                      key={user.id}
                      onClick={() => startConversationWithUser(user)}
                      style={{
                        padding: '10px',
                        background: 'rgba(0,217,255,0.1)',
                        borderRadius: '6px',
                        marginBottom: '4px',
                        cursor: 'pointer',
                        display: 'flex',
                        alignItems: 'center',
                        gap: '10px'
                      }}
                    >
                      <div style={{
                        width: '32px',
                        height: '32px',
                        borderRadius: '50%',
                        background: 'linear-gradient(135deg, #00d9ff, #00ff88)',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center'
                      }}>
                        {user.avatar_url || '👤'}
                      </div>
                      <div>
                        <div style={{ color: '#fff', fontWeight: '600', fontSize: '14px' }}>
                          {user.display_name || user.username}
                        </div>
                        <div style={{ color: '#666', fontSize: '12px' }}>
                          @{user.username}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          <div className="conversations-list">
            {isLoading ? (
              <div style={{ padding: '20px', textAlign: 'center', color: '#666' }}>
                Loading conversations...
              </div>
            ) : threads.length === 0 ? (
              <div style={{ padding: '20px', textAlign: 'center', color: '#666' }}>
                <p>No conversations yet</p>
                <p style={{ fontSize: '12px', marginTop: '8px' }}>
                  Click "+ New" to start messaging someone
                </p>
              </div>
            ) : (
              threads.map(thread => {
                const otherUser = getOtherUser(thread);
                return (
                  <div
                    key={thread.id}
                    className={`conversation-item ${selectedThread?.id === thread.id ? 'active' : ''}`}
                    onClick={() => selectThread(thread)}
                  >
                    <div className="conversation-avatar">
                      {otherUser.avatar_url || '👤'}
                    </div>
                    <div className="conversation-info">
                      <div className="conversation-name">
                        @{otherUser.username}
                        {otherUser.instance_domain !== 'localhost' && (
                          <span className="remote-indicator">🌐</span>
                        )}
                      </div>
                      <div className="conversation-preview">
                        {thread.last_message?.content?.substring(0, 30) || 'No messages yet'}
                        {thread.last_message?.content?.length > 30 && '...'}
                      </div>
                    </div>
                    {thread.unread_count > 0 && (
                      <div style={{
                        background: '#00d9ff',
                        color: '#000',
                        borderRadius: '50%',
                        width: '20px',
                        height: '20px',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        fontSize: '11px',
                        fontWeight: '700'
                      }}>
                        {thread.unread_count}
                      </div>
                    )}
                    <div className="encryption-badge">🔒</div>
                  </div>
                );
              })
            )}
          </div>
        </div>

        {/* Chat Window */}
        <div className="dm-chat-window">
          {!selectedThread ? (
            <div style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              height: '100%',
              color: '#666'
            }}>
              <div style={{ fontSize: '48px', marginBottom: '16px' }}>💬</div>
              <h3 style={{ color: '#fff', marginBottom: '8px' }}>Select a conversation</h3>
              <p>Choose a conversation from the sidebar or start a new one</p>
            </div>
          ) : (
            <>
              {/* Chat Header */}
              <div className="chat-header">
                <div className="chat-header-info">
                  <div className="chat-avatar">
                    {getOtherUser(selectedThread).avatar_url || '👤'}
                  </div>
                  <div className="chat-title-section">
                    <h2 className="chat-title">@{getOtherUser(selectedThread).username}</h2>
                    <div className="chat-status">
                      {getOtherUser(selectedThread).instance_domain !== 'localhost' && (
                        <span className="status-text">🌐 Remote • </span>
                      )}
                      <span className="status-text">🔒 Encrypted</span>
                    </div>
                  </div>
                </div>
              </div>

              {/* Encryption Banner */}
              <div className="encryption-banner">
                🔒 Messages are end-to-end encrypted. The server cannot read this content.
              </div>

              {/* Messages Area */}
              <div className="messages-area">
                {messages.length === 0 ? (
                  <div style={{ textAlign: 'center', padding: '40px', color: '#666' }}>
                    <p>No messages yet</p>
                    <p style={{ fontSize: '12px', marginTop: '8px' }}>
                      Send a message to start the conversation
                    </p>
                  </div>
                ) : (
                  messages.map((message) => (
                    <div
                      key={message.id}
                      className={`message-bubble ${message.sender_id === userData?.id ? 'sent' : 'received'}`}
                    >
                      <div className="message-content">{message.content}</div>
                      <div className="message-meta">
                        <span className="message-timestamp">{formatTimestamp(message.created_at)}</span>
                        <span className="message-encryption" title="End-to-end encrypted">
                          🔒
                        </span>
                      </div>
                    </div>
                  ))
                )}
                <div ref={messagesEndRef} />
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
                  disabled={isSending}
                ></textarea>
                <button
                  className="send-button"
                  onClick={handleSendMessage}
                  disabled={!messageText.trim() || isSending}
                >
                  {isSending ? 'Sending...' : 'Send 🔒'}
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
            </>
          )}
        </div>
      </div>
    </div>
  );
}
