'use client';

import { useState, useEffect } from 'react';
import { useTheme } from '@/components/ui/theme-provider';
import '../styles/HomePage.css';
import { adminApi, postApi, searchApi } from '@/lib/api';

export default function AdminPage({ onNavigate, userData, handleLogout }) {
  /* ================= THEME (from your branch) ================= */
  const { theme, toggleTheme } = useTheme();
  const isDarkMode = theme === 'dark';

  /* ================= STATE (from main branch – unchanged) ================= */
  const [activeTab, setActiveTab] = useState('home');
  const [posts, setPosts] = useState([]);
  const [users, setUsers] = useState([]);
  const [moderationRequests, setModerationRequests] = useState([]);
  const [suspendedUsers, setSuspendedUsers] = useState([]);
  const [adminActions, setAdminActions] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [totalUsers, setTotalUsers] = useState(0);
  const [page, setPage] = useState(0);
  const [actionLoading, setActionLoading] = useState(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [showSearchResults, setShowSearchResults] = useState(false);
  const [searchResults, setSearchResults] = useState([]);
  const [isSearching, setIsSearching] = useState(false);

  const [showBanModal, setShowBanModal] = useState(false);
  const [banTarget, setBanTarget] = useState(null);
  const [banReason, setBanReason] = useState('');

  /* ================= ACCESS CHECK ================= */
  if (userData?.role !== 'admin') {
    return (
      <div style={{
        minHeight: '100vh',
        background: 'var(--bg-primary)',
        color: 'var(--text-primary)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center'
      }}>
        <h2>⛔ Access Denied</h2>
      </div>
    );
  }

  /* ================= EFFECTS ================= */
  useEffect(() => {
    loadTabData();
  }, [activeTab, page]);

  /* ================= DATA LOADERS ================= */

  const loadTabData = async () => {
    setIsLoading(true);
    try {
      if (activeTab === 'home') fetchPosts();
      if (activeTab === 'requests') fetchModerationRequests();
      if (activeTab === 'bans') {
        fetchSuspendedUsers();
        fetchAdminActions();
      }
      if (activeTab === 'users') fetchAllUsers();
    } finally {
      setIsLoading(false);
    }
  };

  const fetchPosts = async () => {
    const feedPosts = await postApi.getPublicFeed(20, 0, false);
    setPosts(feedPosts || []);
  };

  const fetchModerationRequests = async () => {
    const result = await adminApi.getModerationRequests();
    setModerationRequests(result.requests || []);
  };

  const fetchSuspendedUsers = async () => {
    const result = await adminApi.getSuspendedUsers(100, 0);
    setSuspendedUsers(result.users || []);
  };

  const fetchAdminActions = async () => {
    const result = await adminApi.getAdminActions();
    setAdminActions(result.actions || []);
  };

  const fetchAllUsers = async () => {
    const result = await adminApi.getAllUsers(50, page * 50);
    setUsers(result.users || []);
    setTotalUsers(result.total || 0);
  };

  /* ================= HANDLERS ================= */

  const handleDeletePost = async (postId) => {
    await postApi.deletePost(postId);
    setPosts(posts.filter(p => p.id !== postId));
  };

  const formatTimestamp = (iso) => {
    if (!iso) return '';
    const diff = Date.now() - new Date(iso);
    const mins = Math.floor(diff / 60000);
    if (mins < 60) return `${mins}m ago`;
    return new Date(iso).toLocaleDateString();
  };

  /* ================= UI (UNCHANGED FROM MAIN) ================= */

  return (
    <div className="home-container">

      {/* ================= NAVBAR ================= */}
      <nav className="home-nav">

        <div className="nav-left">
          <img src="/logo.png" className="nav-logo-img" />
        </div>

        <div className="nav-center">
          <button className="nav-item" onClick={() => setActiveTab('home')}>🏠 Feed</button>
          <button className="nav-item" onClick={() => setActiveTab('users')}>Users</button>
          <button className="nav-item" onClick={() => setActiveTab('bans')}>Bans</button>
        </div>

        <div className="nav-right">

          {/* 🔥 THEME BUTTON (WORKING) */}
          <button
            onClick={toggleTheme}
            className="nav-btn"
            title="Toggle theme"
          >
            {isDarkMode ? '🌙' : '☀️'}
          </button>

          <button onClick={handleLogout}>Logout</button>
        </div>
      </nav>


      {/* ================= MAIN CONTENT ================= */}
      <main className="home-feed">

        {activeTab === 'home' && posts.map(post => (
          <div key={post.id} className="post">
            <div className="post-content">{post.content}</div>
            <button onClick={() => handleDeletePost(post.id)}>Delete</button>
          </div>
        ))}

        {activeTab === 'users' && users.map(u => (
          <div key={u.id} className="post">{u.username}</div>
        ))}

      </main>

    </div>
  );
}
