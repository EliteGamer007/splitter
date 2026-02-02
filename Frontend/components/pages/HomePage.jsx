'use client';

import React, { useState, useEffect } from 'react';
import '../styles/HomePage.css';
import { postApi, interactionApi, adminApi, searchApi, messageApi } from '@/lib/api';

// Sample posts for demo when no backend posts available
const SAMPLE_POSTS = [
  {
    id: 'sample-1',
    author: 'alice@federate.tech',
    avatar: '👩',
    displayName: 'Alice Chen',
    handle: '@alice',
    timestamp: '2h ago',
    content: 'Just deployed a new federated instance! The decentralization is working beautifully. 🚀',
    replies: 12,
    boosts: 45,
    likes: 128,
    local: true,
    visibility: 'public'
  },
  {
    id: 'sample-2',
    author: 'bob@community.social',
    avatar: '👨',
    displayName: 'Bob Smith',
    handle: '@bob',
    timestamp: '4h ago',
    content: 'Love the transparency in this federated model. No hidden algorithms, just real human connections. 💙',
    replies: 8,
    boosts: 32,
    likes: 95,
    local: false,
    visibility: 'public'
  },
  {
    id: 'sample-3',
    author: 'charlie@tech-minds.io',
    avatar: '👨‍💻',
    displayName: 'Charlie Dev',
    handle: '@charlie',
    timestamp: '6h ago',
    content: 'Implemented end-to-end encryption for DMs. Your messages are truly private now. 🔐',
    replies: 24,
    boosts: 67,
    likes: 234,
    local: false,
    visibility: 'public'
  }
];

import { useTheme } from '@/components/ui/theme-provider';

export default function HomePage({ onNavigate, userData, updateUserData, handleLogout }) {
  const { theme, toggleTheme } = useTheme();



  const isDarkMode = theme === 'dark';
  const [activeTab, setActiveTab] = useState('home');
  const [newPostText, setNewPostText] = useState('');
  const [newPostVisibility, setNewPostVisibility] = useState('public');
  const [posts, setPosts] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isPosting, setIsPosting] = useState(false);
  const [error, setError] = useState(null);
  const [selectedFile, setSelectedFile] = useState(null);
  const [previewUrl, setPreviewUrl] = useState(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [showSearchResults, setShowSearchResults] = useState(false);
  const [searchResults, setSearchResults] = useState([]);
  const [isSearching, setIsSearching] = useState(false);
  const [followingUsers, setFollowingUsers] = useState(new Set());
  const [followLoading, setFollowLoading] = useState(new Set());

  // Edit/Delete state
  const [editingPostId, setEditingPostId] = useState(null);
  const [editText, setEditText] = useState('');
  const [deleteConfirmId, setDeleteConfirmId] = useState(null);

  // Fetch posts on mount and when tab changes
  useEffect(() => {
    fetchPosts();
  }, [activeTab]);

  // Fetch current user's following list on mount
  useEffect(() => {
    const loadFollowingList = async () => {
      if (!userData?.id) return;

      try {
        const { followApi } = await import('@/lib/api');
        const following = await followApi.getFollowing(userData.id);

        // Create a Set of user IDs that the current user is following
        const followingIds = new Set(
          (following || []).map(user => user.id)
        );
        setFollowingUsers(followingIds);
        console.log('Loaded following list:', followingIds);
      } catch (err) {
        console.error('Failed to load following list:', err);
      }
    };

    loadFollowingList();
  }, [userData?.id]);

  // Search users
  const handleSearch = async () => {
    if (searchQuery.length < 2) return;

    setIsSearching(true);
    try {
      const result = await searchApi.searchUsers(searchQuery);
      console.log('Search results:', result);
      setSearchResults(result.users || []);
      if (result.users && result.users.length > 0) {
        console.log('First user object:', result.users[0]);
      }
      setShowSearchResults(true);
    } catch (err) {
      console.error('Search failed:', err);
      setSearchResults([]);
    } finally {
      setIsSearching(false);
    }
  };

  // Debounced search
  useEffect(() => {
    if (searchQuery.length >= 2) {
      const timer = setTimeout(handleSearch, 300);
      return () => clearTimeout(timer);
    } else {
      setShowSearchResults(false);
      setSearchResults([]);
    }
  }, [searchQuery]);

  // Start DM with user
  const startDMWithUser = async (user) => {
    try {
      await messageApi.startConversation(user.id);
      setShowSearchResults(false);
      setSearchQuery('');
      onNavigate('dm', { selectedUser: user });
    } catch (err) {
      console.error('Failed to start conversation:', err);
      alert('Failed to start conversation: ' + err.message);
    }
  };

  // Follow/Unfollow user
  const handleFollowToggle = async (userId) => {
    console.log('Follow toggle called for userId:', userId);
    const isFollowing = followingUsers.has(userId);

    // Add to loading set
    setFollowLoading(prev => new Set(prev).add(userId));

    try {
      const { followApi } = await import('@/lib/api');

      if (isFollowing) {
        console.log('Unfollowing user:', userId);
        await followApi.unfollowUser(userId);
        setFollowingUsers(prev => {
          const next = new Set(prev);
          next.delete(userId);
          return next;
        });
      } else {
        console.log('Following user:', userId);
        await followApi.followUser(userId);
        setFollowingUsers(prev => new Set(prev).add(userId));
      }
      console.log('Follow operation successful');
    } catch (err) {
      console.error('Follow operation failed:', err);
      alert(`Failed to ${isFollowing ? 'unfollow' : 'follow'} user: ${err.message}`);
    } finally {
      // Remove from loading set
      setFollowLoading(prev => {
        const next = new Set(prev);
        next.delete(userId);
        return next;
      });
    }
  };

  const fetchPosts = async () => {
    setIsLoading(true);
    setError(null);
    try {
      let feedPosts;
      const token = typeof window !== 'undefined' ? localStorage.getItem('jwt_token') : null;

      if (activeTab === 'home') {
        // Home tab: authenticated feed
        if (token) {
          try {
            feedPosts = await postApi.getFeed(20, 0);
          } catch (authErr) {
            feedPosts = await postApi.getPublicFeed(20, 0, false);
          }
        } else {
          feedPosts = await postApi.getPublicFeed(20, 0, false);
        }
      } else if (activeTab === 'local') {
        // Local tab: local_only = true
        feedPosts = await postApi.getPublicFeed(20, 0, true);
      } else if (activeTab === 'federated') {
        // Federated tab: all public posts (local_only = false)
        feedPosts = await postApi.getPublicFeed(20, 0, false);
      }

      if (feedPosts && feedPosts.length > 0) {
        const transformedPosts = feedPosts.map(post => ({
          id: post.id,
          author: post.username ? `${post.username}@localhost` : `${post.author_did?.split(':').pop() || 'unknown'}@local`,
          authorId: post.author_id || post.user_id,
          avatar: post.avatar_url || '👤',
          displayName: post.username || post.author_did?.split(':').pop() || 'Unknown',
          handle: `@${post.username || post.author_did?.split(':').pop() || 'unknown'}`,
          timestamp: formatTimestamp(post.created_at),
          createdAt: post.created_at,
          updatedAt: post.updated_at,
          content: post.content,
          replies: post.reply_count || 0,
          boosts: post.repost_count || 0,
          likes: post.like_count || 0,
          local: post.is_local !== undefined ? post.is_local : true,
          visibility: post.visibility || 'public',
          liked: post.liked || false,
          reposted: post.reposted || false,
          bookmarked: post.bookmarked || false,
          imageUrl: post.media?.[0]?.media_url || null
        }));
        setPosts(transformedPosts);
      } else {
        setPosts(SAMPLE_POSTS);
      }
    } catch (err) {
      console.error('Failed to fetch posts:', err);
      setPosts(SAMPLE_POSTS);
    } finally {
      setIsLoading(false);
    }
  };

  const formatTimestamp = (isoString) => {
    if (!isoString) return 'unknown';
    const date = new Date(isoString);
    const now = new Date();
    const diffMs = now - date;
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return 'now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    return `${diffDays}d ago`;
  };

  const handlePostCreate = async () => {
    // allow text OR image OR both
    if (!newPostText.trim() && !selectedFile) return;

    setIsPosting(true);

    try {
      const newPost = await postApi.createPost(
        newPostText.trim(),          // always send content
        newPostVisibility,
        selectedFile || undefined   // always send file if exists
      );

      const transformedPost = {
        id: newPost.id,
        author: `${userData.username}@${userData.server}`,
        authorId: userData.id,
        avatar: userData.avatar,
        displayName: userData.displayName,
        handle: `@${userData.username}`,
        timestamp: 'now',
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        content: newPost.content,
        replies: 0,
        boosts: 0,
        likes: 0,
        local: true,
        visibility: newPostVisibility,
        imageUrl: newPost.media?.[0]?.media_url || null
      };

      setPosts(prev => [transformedPost, ...prev]);

      // reset composer
      setNewPostText('');
      setSelectedFile(null);
      setPreviewUrl(null);

    } catch (err) {
      setError('Failed to create post: ' + err.message);
    } finally {
      setIsPosting(false);
    }
  };

  // Edit post
  const handleEditStart = (post) => {
    setEditingPostId(post.id);
    setEditText(post.content);
  };

  const handleEditCancel = () => {
    setEditingPostId(null);
    setEditText('');
  };

  const handleEditSave = async (postId) => {
    if (!editText.trim()) return;

    try {
      await postApi.updatePost(postId, editText);
      setPosts(prev => prev.map(p =>
        p.id === postId
          ? { ...p, content: editText, updatedAt: new Date().toISOString() }
          : p
      ));
      setEditingPostId(null);
      setEditText('');
    } catch (err) {
      alert('Failed to update post: ' + err.message);
    }
  };

  // Delete post
  const handleDeleteConfirm = async (postId) => {
    try {
      await postApi.deletePost(postId);
      setPosts(prev => prev.filter(p => p.id !== postId));
      setDeleteConfirmId(null);
    } catch (err) {
      alert('Failed to delete post: ' + err.message);
    }
  };

  const handleLike = async (postId) => {
    try {
      const post = posts.find(p => p.id === postId);
      if (post?.liked) {
        await interactionApi.unlikePost(postId);
        setPosts(prev => prev.map(p =>
          p.id === postId ? { ...p, liked: false, likes: p.likes - 1 } : p
        ));
      } else {
        await interactionApi.likePost(postId);
        setPosts(prev => prev.map(p =>
          p.id === postId ? { ...p, liked: true, likes: p.likes + 1 } : p
        ));
      }
    } catch (err) {
      console.error('Like failed:', err);
    }
  };

  const handleRepost = async (postId) => {
    try {
      const post = posts.find(p => p.id === postId);
      if (post?.reposted) {
        await interactionApi.unrepostPost(postId);
        setPosts(prev => prev.map(p =>
          p.id === postId ? { ...p, reposted: false, boosts: p.boosts - 1 } : p
        ));
      } else {
        await interactionApi.repostPost(postId);
        setPosts(prev => prev.map(p =>
          p.id === postId ? { ...p, reposted: true, boosts: p.boosts + 1 } : p
        ));
      }
    } catch (err) {
      console.error('Repost failed:', err);
    }
  };

  // Check if post was edited
  const isEdited = (post) => {
    if (!post.createdAt || !post.updatedAt) return false;
    return new Date(post.updatedAt).getTime() - new Date(post.createdAt).getTime() > 1000;
  };

  // Check if current user owns the post
  const isOwnPost = (post) => {
    return post.authorId === userData?.id ||
      post.handle === `@${userData?.username}` ||
      post.author?.startsWith(userData?.username);
  };

  const getVisibilityIcon = (visibility) => {
    switch (visibility) {
      case 'followers': return '👥';
      case 'circle': return '🔒';
      default: return '🌐';
    }
  };

  return (
    <div className="home-container">
      {/* Top Navigation */}
      <nav className="home-nav">
        <div className="nav-left">
          <h1 className="nav-logo">🌐 SPLITTER</h1>
        </div>
        <div className="nav-center">
          <button
            className={`nav-item ${activeTab === 'home' ? 'active' : ''}`}
            onClick={() => setActiveTab('home')}
          >
            Home
          </button>
          <button
            className={`nav-item ${activeTab === 'local' ? 'active' : ''}`}
            onClick={() => setActiveTab('local')}
          >
            Local
          </button>
          <button
            className={`nav-item ${activeTab === 'federated' ? 'active' : ''}`}
            onClick={() => setActiveTab('federated')}
          >
            Federated
          </button>
        </div>
        <div className="nav-right">
          <div style={{ position: 'relative' }}>
            <input
              type="text"
              placeholder="Search users..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onFocus={() => searchQuery.length >= 2 && setShowSearchResults(true)}
              className="nav-search"
              style={{
                minWidth: showSearchResults ? '400px' : '300px',
                width: showSearchResults ? '450px' : '300px',
                transition: 'all 0.3s ease'
              }}
            />
            {/* Search Results Dropdown */}
            {showSearchResults && (
              <div style={{
                position: 'absolute',
                top: '100%',
                left: 0,
                right: 0,
                background: '#1a1a2e',
                border: '1px solid #333',
                borderRadius: '8px',
                marginTop: '4px',
                maxHeight: '300px',
                overflowY: 'auto',
                zIndex: 1000,
                boxShadow: '0 4px 20px rgba(0,0,0,0.5)'
              }}>
                {isSearching ? (
                  <div style={{ padding: '16px', textAlign: 'center', color: '#666' }}>
                    Searching...
                  </div>
                ) : searchResults.length === 0 ? (
                  <div style={{ padding: '16px', textAlign: 'center', color: '#666' }}>
                    No users found
                  </div>
                ) : (
                  searchResults.map(user => (
                    <div
                      key={user.id}
                      style={{
                        padding: '12px 16px',
                        borderBottom: '1px solid #333',
                        display: 'flex',
                        alignItems: 'center',
                        gap: '12px',
                        transition: 'background 0.2s'
                      }}
                    >
                      <div
                        onClick={() => {
                          setShowSearchResults(false);
                          setSearchQuery('');
                          onNavigate('profile', { userId: user.id });
                        }}
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          gap: '12px',
                          flex: 1,
                          cursor: 'pointer',
                        }}
                        onMouseEnter={(e) => e.currentTarget.style.opacity = '0.8'}
                        onMouseLeave={(e) => e.currentTarget.style.opacity = '1'}
                      >
                        <div style={{
                          width: '40px',
                          height: '40px',
                          borderRadius: '50%',
                          background: 'linear-gradient(135deg, #00d9ff, #00ff88)',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          fontSize: '18px'
                        }}>
                          {user.avatar_url || '👤'}
                        </div>
                        <div style={{ flex: 1 }}>
                          <div style={{ color: '#fff', fontWeight: '600' }}>
                            {user.display_name || user.username}
                          </div>
                          <div style={{ color: '#666', fontSize: '12px' }}>
                            @{user.username}@{user.instance_domain}
                          </div>
                        </div>
                      </div>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          console.log('Follow button clicked for user:', user);
                          handleFollowToggle(user.id);
                        }}
                        disabled={followLoading.has(user.id)}
                        style={{
                          padding: '6px 16px',
                          background: followingUsers.has(user.id) ? 'rgba(0,255,136,0.2)' : 'rgba(0,217,255,0.2)',
                          border: `1px solid ${followingUsers.has(user.id) ? '#00ff88' : '#00d9ff'}`,
                          color: followingUsers.has(user.id) ? '#00ff88' : '#00d9ff',
                          borderRadius: '4px',
                          cursor: followLoading.has(user.id) ? 'not-allowed' : 'pointer',
                          fontSize: '12px',
                          minWidth: '80px',
                          opacity: followLoading.has(user.id) ? 0.6 : 1
                        }}
                      >
                        {followLoading.has(user.id) ? '...' : followingUsers.has(user.id) ? '✓ Following' : 'Follow'}
                      </button>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          startDMWithUser(user);
                        }}
                        style={{
                          padding: '6px 12px',
                          background: 'rgba(0,217,255,0.2)',
                          border: '1px solid #00d9ff',
                          color: '#00d9ff',
                          borderRadius: '4px',
                          cursor: 'pointer',
                          fontSize: '12px'
                        }}
                      >
                        💬 DM
                      </button>
                    </div>
                  ))
                )}
                {searchResults.length > 0 && (
                  <div
                    style={{
                      padding: '8px 16px',
                      textAlign: 'center',
                      borderTop: '1px solid #333'
                    }}
                  >
                    <button
                      onClick={() => setShowSearchResults(false)}
                      style={{
                        background: 'none',
                        border: 'none',
                        color: '#666',
                        cursor: 'pointer',
                        fontSize: '12px'
                      }}
                    >
                      Close
                    </button>
                  </div>
                )}
              </div>
            )}
          </div>
          <button
            className="nav-btn-profile"
            onClick={() => onNavigate('profile')}
          >
            Profile
          </button>
          <button
            className="nav-btn-profile"
            onClick={toggleTheme}
            title={isDarkMode ? 'Switch to Light Mode' : 'Switch to Dark Mode'}
            style={{
              marginLeft: '10px',
              padding: '8px 12px',
              background: isDarkMode ? 'rgba(0, 217, 255, 0.1)' : 'rgba(100, 100, 100, 0.1)',
              border: `1px solid ${isDarkMode ? '#00d9ff' : '#666'}`,
              color: isDarkMode ? '#00d9ff' : '#333',
              borderRadius: '6px',
              cursor: 'pointer'
            }}
          >
            {theme === "dark" ? "🌞" : "🌙"}
          </button>
          {handleLogout && (
            <button
              className="nav-btn-profile"
              onClick={handleLogout}
              title="Logout"
              style={{
                marginLeft: '10px',
                padding: '8px 12px',
                background: 'rgba(255, 68, 68, 0.1)',
                border: '1px solid #ff4444',
                color: '#ff4444',
                borderRadius: '6px',
                cursor: 'pointer'
              }}
            >
              <svg viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" height="16" width="16">
                <path d="M1.728 8c0 0.20793333333333333 0.08259999999999999 0.4074 0.22959999999999997 0.5544 0.14706666666666665 0.147 0.3464666666666667 0.22959999999999997 0.5544 0.22959999999999997h5.9505333333333335L6.6593333333333335 10.579333333333333c-0.07339999999999999 0.07293333333333332 -0.1317333333333333 0.15966666666666665 -0.1716 0.2551333333333333 -0.0398 0.09559999999999999 -0.06026666666666666 0.19806666666666664 -0.06026666666666666 0.3015333333333333s0.020466666666666668 0.20593333333333333 0.06026666666666666 0.3015333333333333c0.03986666666666666 0.09546666666666666 0.09819999999999998 0.18219999999999997 0.1716 0.2551333333333333 0.07293333333333332 0.07346666666666667 0.15966666666666665 0.13179999999999997 0.2551333333333333 0.1716 0.09559999999999999 0.0398 0.19806666666666664 0.06026666666666666 0.3015333333333333 0.06026666666666666s0.20593333333333333 -0.020466666666666668 0.3015333333333333 -0.06026666666666666c0.09546666666666666 -0.0398 0.18219999999999997 -0.09813333333333332 0.2551333333333333 -0.1716l3.1359999999999997 -3.1359999999999997c0.07133333333333333 -0.0746 0.12726666666666664 -0.16246666666666665 0.1646 -0.2587333333333333 0.0784 -0.19093333333333332 0.0784 -0.40493333333333337 0 -0.5958666666666667 -0.03733333333333333 -0.09626666666666667 -0.09326666666666666 -0.18413333333333332 -0.1646 -0.2587333333333333l-3.1359999999999997 -3.1359999999999997c-0.07306666666666667 -0.07306666666666667 -0.15986666666666666 -0.13106666666666666 -0.2554666666666666 -0.1706 -0.09546666666666666 -0.039599999999999996 -0.1978 -0.059933333333333325 -0.30119999999999997 -0.059933333333333325 -0.10339999999999999 0 -0.20573333333333332 0.02033333333333333 -0.30119999999999997 0.059933333333333325 -0.09559999999999999 0.03953333333333333 -0.1824 0.09753333333333333 -0.2554666666666666 0.1706 -0.07306666666666667 0.07313333333333333 -0.13106666666666666 0.15993333333333332 -0.17066666666666666 0.25539999999999996 -0.039466666666666664 0.09553333333333333 -0.059866666666666665 0.19786666666666666 -0.059866666666666665 0.3012666666666667 0 0.10339999999999999 0.020399999999999998 0.20573333333333332 0.059866666666666665 0.3012666666666667 0.039599999999999996 0.09546666666666666 0.09759999999999999 0.18226666666666663 0.17066666666666666 0.25539999999999996l1.8032 1.7953333333333332H2.5119999999999996c-0.20793333333333333 0 -0.4073333333333333 0.08266666666666667 -0.5544 0.22966666666666663 -0.147 0.147 -0.22959999999999997 0.34639999999999993 -0.22959999999999997 0.5543333333333333ZM11.919999999999998 0.15999999999999998H4.08c-0.6237999999999999 0 -1.222 0.24779999999999996 -1.6631333333333331 0.6888666666666667 -0.4410666666666666 0.44113333333333327 -0.6888666666666667 1.0393333333333332 -0.6888666666666667 1.6631333333333331v2.352c0 0.20793333333333333 0.08259999999999999 0.4073333333333333 0.22959999999999997 0.5544 0.14706666666666665 0.147 0.3464666666666667 0.22959999999999997 0.5544 0.22959999999999997s0.4073333333333333 -0.08259999999999999 0.5544 -0.22959999999999997c0.147 -0.14706666666666665 0.22959999999999997 -0.3464666666666667 0.22959999999999997 -0.5544V2.5119999999999996c0 -0.20793333333333333 0.08259999999999999 -0.4073333333333333 0.22959999999999997 -0.5544 0.14706666666666665 -0.147 0.3464666666666667 -0.22959999999999997 0.5544 -0.22959999999999997h7.84c0.20793333333333333 0 0.4074 0.08259999999999999 0.5544 0.22959999999999997 0.147 0.14706666666666665 0.22959999999999997 0.3464666666666667 0.22959999999999997 0.5544v10.975999999999999c0 0.20793333333333333 -0.08259999999999999 0.4073333333333333 -0.22959999999999997 0.5544s-0.3464666666666667 0.22959999999999997 -0.5544 0.22959999999999997H4.08c-0.20793333333333333 0 -0.4073333333333333 -0.08259999999999999 -0.5544 -0.22959999999999997 -0.147 -0.147 -0.22959999999999997 -0.3464666666666667 -0.22959999999999997 -0.5544v-2.352c0 -0.20793333333333333 -0.08259999999999999 -0.4073333333333333 -0.22959999999999997 -0.5543333333333333 -0.14706666666666665 -0.147 -0.3464666666666667 -0.22966666666666663 -0.5544 -0.22966666666666663s-0.4073333333333333 0.08266666666666667 -0.5544 0.22966666666666663c-0.147 0.147 -0.22959999999999997 0.34639999999999993 -0.22959999999999997 0.5543333333333333v2.352c0 0.6237333333333333 0.24779999999999996 1.222 0.6888666666666667 1.6631333333333331 0.44113333333333327 0.4410666666666666 1.0393333333333332 0.6888666666666667 1.6631333333333331 0.6888666666666667h7.84c0.6237333333333333 0 1.222 -0.24779999999999996 1.6631333333333331 -0.6888666666666667 0.4410666666666666 -0.44113333333333327 0.6888666666666667 -1.0393999999999999 0.6888666666666667 -1.6631333333333331V2.5119999999999996c0 -0.6237999999999999 -0.24779999999999996 -1.222 -0.6888666666666667 -1.6631333333333331C13.142 0.4078 12.543733333333332 0.15999999999999998 11.919999999999998 0.15999999999999998Z" fill={isDarkMode ? '#ff4444' : '#ff4444'} strokeWidth="0.6667"></path>
              </svg>
            </button>
          )}
        </div>
      </nav>

      {/* Main Layout */}
      <div className="home-layout">
        {/* Left Sidebar */}
        <aside className="home-sidebar">
          <div className="sidebar-section">
            <h3 className="sidebar-title">Navigation</h3>
            <div className="sidebar-links">
              <button
                className="sidebar-link active"
                onClick={() => setActiveTab('home')}
                style={{ textAlign: 'left', width: '100%' }}
              >
                <span>Home</span>
              </button>
              <button
                className="sidebar-link"
                onClick={() => onNavigate('dm')}
                style={{ textAlign: 'left', width: '100%' }}
              >
                <span>Messages 🔒</span>
              </button>
              <button
                className="sidebar-link"
                onClick={() => onNavigate('security')}
                style={{ textAlign: 'left', width: '100%' }}
              >
                <span>Security</span>
              </button>
            </div>
          </div>

          <div className="sidebar-section">
            <h3 className="sidebar-title">Your Profile</h3>
            <div className="sidebar-profile">
              <div className="profile-avatar">{userData.avatar}</div>
              <div className="profile-info">
                <p className="profile-name">{userData.displayName}</p>
                <p className="profile-handle">@{userData.username}@{userData.server}</p>
                <p className="profile-stats">
                  <strong>{userData.following}</strong> Following • <strong>{userData.followers}</strong> Followers
                </p>
              </div>
              <button
                className="sidebar-btn"
                onClick={() => onNavigate('security')}
              >
                Settings
              </button>
            </div>
          </div>

          <div className="sidebar-section">
            <h3 className="sidebar-title">Server Info</h3>
            <div className="sidebar-info">
              <div className="info-item">
                <span className="info-label">Server</span>
                <span className="info-value">{userData.server || userData.instance_domain || 'localhost'}</span>
              </div>
              <div className="info-item">
                <span className="info-label">Your Role</span>
                <span className="info-value" style={{
                  color: userData.role === 'admin' ? '#ff4444' :
                    userData.role === 'moderator' ? '#00d9ff' : '#00ff88'
                }}>
                  {userData.role === 'admin' ? 'Admin' :
                    userData.role === 'moderator' ? 'Moderator' : 'User'}
                </span>
              </div>
              <div className="info-item">
                <span className="info-label">Reputation</span>
                <span className="info-value">🟢 Trusted</span>
              </div>
              <div className="info-item">
                <span className="info-label">Federation</span>
                <span className="info-value">🌐 Open</span>
              </div>
            </div>
          </div>
        </aside>

        {/* Main Feed */}
        <main className="home-feed">
          {/* Error display */}
          {error && (
            <div style={{
              background: 'rgba(255, 68, 68, 0.1)',
              border: '1px solid #ff4444',
              color: '#ff4444',
              padding: '12px',
              borderRadius: '8px',
              marginBottom: '16px'
            }}>
              ⚠️ {error}
            </div>
          )}

          {/* Composer */}
          <div className="feed-composer">
            <div className="composer-header">
              <h2>What's happening? 🌐</h2>
            </div>
            <div className="composer-body">
              <textarea
                className="composer-textarea"
                placeholder="Share your thoughts with the federated network..."
                value={newPostText}
                onChange={(e) => setNewPostText(e.target.value)}
                maxLength="500"
                disabled={isPosting}
              />

              {/* Media Preview */}
              {previewUrl && (
                <div style={{ position: 'relative', margin: '10px 0', maxWidth: '100%' }}>
                  <img
                    src={previewUrl}
                    alt="Preview"
                    style={{
                      maxHeight: '300px',
                      maxWidth: '100%',
                      borderRadius: '8px',
                      objectFit: 'cover'
                    }}
                  />
                  <button
                    onClick={() => {
                      setSelectedFile(null);
                      setPreviewUrl(null);
                      // Reset file input
                      const fileInput = document.getElementById('post-media-input');
                      if (fileInput) fileInput.value = '';
                    }}
                    style={{
                      position: 'absolute',
                      top: '8px',
                      right: '8px',
                      background: 'rgba(0,0,0,0.6)',
                      color: 'white',
                      border: 'none',
                      borderRadius: '50%',
                      width: '28px',
                      height: '28px',
                      cursor: 'pointer',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: '16px'
                    }}
                  >
                    ×
                  </button>
                </div>
              )}

              <div className="composer-footer">
                <div className="composer-info">
                  <span className="char-count">
                    {newPostText.length}/500
                  </span>
                  {/* Visibility Selector */}
                  <select
                    value={newPostVisibility}
                    onChange={(e) => setNewPostVisibility(e.target.value)}
                    style={{
                      padding: '4px 8px',
                      background: '#1a1a2e',
                      border: '1px solid #333',
                      color: '#00d9ff',
                      borderRadius: '4px',
                      cursor: 'pointer',
                      fontSize: '12px'
                    }}
                  >
                    <option value="public">🌐 Public</option>
                    <option value="followers">👥 Followers</option>
                    <option value="circle">🔒 Circle</option>
                  </select>
                </div>
                <div className="composer-actions">
                  <input
                    type="file"
                    id="post-media-input"
                    accept="image/png, image/jpeg, image/gif"
                    style={{ display: 'none' }}
                    onChange={(e) => {
                      if (e.target.files && e.target.files[0]) {
                        const file = e.target.files[0];

                        // Validate size
                        if (file.size > 5 * 1024 * 1024) {
                          alert('File size exceeds 5MB limit.');
                          return;
                        }

                        setSelectedFile(file);
                        const url = URL.createObjectURL(file);
                        setPreviewUrl(url);
                      }
                    }}
                  />
                  <button
                    className="composer-btn-media"
                    onClick={() => document.getElementById('post-media-input').click()}
                    title="Attach media (max 5MB)"
                    disabled={isPosting}
                    style={{
                      background: 'none',
                      border: 'none',
                      cursor: 'pointer',
                      opacity: isPosting ? 0.5 : 1
                    }}
                  >
                    🖼️ Media
                  </button>
                  <button
                    className={`composer-btn-post ${(!newPostText.trim() && !selectedFile) || isPosting ? 'disabled' : ''}`}
                    onClick={handlePostCreate}
                    disabled={(!newPostText.trim() && !selectedFile) || isPosting}
                  >
                    {isPosting ? 'Posting...' : 'Post'}
                  </button>
                </div>
              </div>
            </div>
          </div>

          {/* Feed Divider */}
          <div className="feed-divider" />

          {/* Loading indicator */}
          {isLoading && (
            <div style={{ textAlign: 'center', padding: '20px' }}>
              Loading posts...
            </div>
          )}

          {/* Posts */}
          <div className="feed-posts">
            {posts.map(post => (
              <article key={post.id} className={`post ${post.local ? 'local' : 'remote'}`}>
                <div className="post-header">
                  <div className="post-author" style={{ cursor: 'pointer' }} onClick={() => onNavigate('profile')}>
                    <span className="post-avatar">{post.avatar}</span>
                    <div className="post-meta">
                      <div className="post-name-line">
                        <strong>{post.displayName}</strong>
                        <span className="post-handle">{post.handle}</span>
                        {post.local && <span className="post-badge local">Local</span>}
                        {!post.local && <span className="post-badge remote">🌐 Remote</span>}
                        {/* Edited Badge */}
                        {isEdited(post) && (
                          <span
                            className="post-badge"
                            style={{ background: 'rgba(255,170,0,0.2)', color: '#ffaa00' }}
                            title={`Edited: ${new Date(post.updatedAt).toLocaleString()}`}
                          >
                            ✏️ Edited
                          </span>
                        )}
                      </div>
                      <span className="post-time">{post.timestamp}</span>
                    </div>
                  </div>
                  {post.visibility && post.visibility !== 'public' && (
                    <span className="post-visibility" style={{ color: '#888', fontSize: '12px' }}>
                      {getVisibilityIcon(post.visibility)} {post.visibility === 'followers' ? 'Followers Only' : 'Circle Only'}
                    </span>
                  )}
                </div>

                {/* Post Content or Edit Form */}
                {editingPostId === post.id ? (
                  <div style={{ margin: '12px 0' }}>
                    <textarea
                      value={editText}
                      onChange={(e) => setEditText(e.target.value)}
                      maxLength="500"
                      style={{
                        width: '100%',
                        minHeight: '80px',
                        padding: '12px',
                        background: '#1a1a2e',
                        border: '1px solid #00d9ff',
                        borderRadius: '8px',
                        color: '#fff',
                        resize: 'vertical'
                      }}
                    />
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '8px' }}>
                      <span style={{ color: '#666', fontSize: '12px' }}>{editText.length}/500</span>
                      <div style={{ display: 'flex', gap: '8px' }}>
                        <button
                          onClick={handleEditCancel}
                          style={{
                            padding: '6px 12px',
                            background: 'rgba(255,68,68,0.1)',
                            border: '1px solid #ff4444',
                            color: '#ff4444',
                            borderRadius: '4px',
                            cursor: 'pointer'
                          }}
                        >
                          Cancel
                        </button>
                        <button
                          onClick={() => handleEditSave(post.id)}
                          style={{
                            padding: '6px 12px',
                            background: 'rgba(0,255,136,0.2)',
                            border: '1px solid #00ff88',
                            color: '#00ff88',
                            borderRadius: '4px',
                            cursor: 'pointer'
                          }}
                        >
                          Save
                        </button>
                      </div>
                    </div>
                  </div>
                ) : (
                  <div
                    className="post-content"
                    style={{ cursor: 'pointer' }}
                    onClick={() => onNavigate('thread')}
                  >
                    {post.content}
                    {post.imageUrl && (
                      <div style={{ marginTop: '10px' }}>
                        <img
                          src={`http://localhost:8000${post.imageUrl}`}
                          alt="Post attachment"
                          style={{
                            maxWidth: '100%',
                            maxHeight: '400px',
                            borderRadius: '8px',
                            objectFit: 'cover'
                          }}
                        />
                      </div>
                    )}
                  </div>
                )}

                {/* Delete Confirmation */}
                {deleteConfirmId === post.id && (
                  <div style={{
                    background: 'rgba(255,68,68,0.1)',
                    border: '1px solid #ff4444',
                    borderRadius: '8px',
                    padding: '12px',
                    margin: '12px 0',
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center'
                  }}>
                    <span style={{ color: '#ff4444' }}>⚠️ Delete this post?</span>
                    <div style={{ display: 'flex', gap: '8px' }}>
                      <button
                        onClick={() => setDeleteConfirmId(null)}
                        style={{
                          padding: '6px 12px',
                          background: 'transparent',
                          border: '1px solid #666',
                          color: '#666',
                          borderRadius: '4px',
                          cursor: 'pointer'
                        }}
                      >
                        Cancel
                      </button>
                      <button
                        onClick={() => handleDeleteConfirm(post.id)}
                        style={{
                          padding: '6px 12px',
                          background: '#ff4444',
                          border: 'none',
                          color: '#fff',
                          borderRadius: '4px',
                          cursor: 'pointer'
                        }}
                      >
                        Delete
                      </button>
                    </div>
                  </div>
                )}

                <div className="post-actions">
                  <button
                    className="post-action"
                    onClick={() => onNavigate('thread')}
                  >
                    <span className="action-icon">💬</span>
                    <span className="action-count">{post.replies}</span>
                  </button>
                  <button
                    className={`post-action ${post.reposted ? 'active' : ''}`}
                    onClick={() => handleRepost(post.id)}
                    style={post.reposted ? { color: '#00d9ff' } : {}}
                  >
                    <span className="action-icon">🚀</span>
                    <span className="action-count">{post.boosts}</span>
                  </button>
                  <button
                    className={`post-action ${post.liked ? 'active' : ''}`}
                    onClick={() => handleLike(post.id)}
                    style={post.liked ? { color: '#ff4444' } : {}}
                  >
                    <span className="action-icon">{post.liked ? '❤️' : '🤍'}</span>
                    <span className="action-count">{post.likes}</span>
                  </button>

                  {/* Edit/Delete buttons for own posts */}
                  {isOwnPost(post) && !editingPostId && !deleteConfirmId && (
                    <>
                      <button
                        className="post-action"
                        onClick={() => handleEditStart(post)}
                        title="Edit post"
                        style={{ marginLeft: 'auto' }}
                      >
                        <span className="action-icon">✏️</span>
                      </button>
                      <button
                        className="post-action"
                        onClick={() => setDeleteConfirmId(post.id)}
                        title="Delete post"
                        style={{ color: '#ff4444' }}
                      >
                        <span className="action-icon">🗑️</span>
                      </button>
                    </>
                  )}

                  {!isOwnPost(post) && (
                    <button className="post-action post-action-delete">
                      <span className="action-icon">⋯</span>
                    </button>
                  )}
                </div>
              </article>
            ))}
          </div>
        </main>

        {/* Right Sidebar */}
        <aside className="home-trends">
          <div className="trends-section">
            <h3 className="trends-title">Trending Topics</h3>
            <div className="trends-list">
              <a href="#" className="trend-item">
                <div className="trend-name">#Decentralization</div>
                <div className="trend-count">2.4K posts</div>
              </a>
              <a href="#" className="trend-item">
                <div className="trend-name">#Federation</div>
                <div className="trend-count">1.8K posts</div>
              </a>
              <a href="#" className="trend-item">
                <div className="trend-name">#PrivacyFirst</div>
                <div className="trend-count">942 posts</div>
              </a>
              <a href="#" className="trend-item">
                <div className="trend-name">#OpenSource</div>
                <div className="trend-count">3.1K posts</div>
              </a>
            </div>
          </div>

          <div className="trends-section">
            <h3 className="trends-title">ℹ️ About This Network</h3>
            <p className="trends-description">
              This is a decentralized social network powered by federation. Your identity is your own, your server is your choice, and your data is encrypted.
            </p>
          </div>

          {/* Admin Panel - Only for admins/moderators */}
          {(userData.role === 'admin' || userData.role === 'moderator') && (
            <div className="trends-section">
              <h3 className="trends-title">Admin Panel</h3>
              <button
                onClick={() => onNavigate('moderation')}
                style={{
                  width: '100%',
                  padding: '10px',
                  background: 'rgba(0, 217, 255, 0.1)',
                  border: '1px solid #00d9ff',
                  color: '#00d9ff',
                  borderRadius: '6px',
                  cursor: 'pointer',
                  marginBottom: '8px',
                  fontWeight: '600',
                  transition: 'all 0.3s ease'
                }}
                className="trend-item"
              >
                Moderation Queue
              </button>
              <button
                onClick={() => onNavigate('federation')}
                style={{
                  width: '100%',
                  padding: '10px',
                  background: 'rgba(0, 217, 255, 0.1)',
                  border: '1px solid #00d9ff',
                  color: '#00d9ff',
                  borderRadius: '6px',
                  cursor: 'pointer',
                  fontWeight: '600',
                  transition: 'all 0.3s ease'
                }}
                className="trend-item"
              >
                Federation Inspector
              </button>
              {userData.role === 'admin' && (
                <button
                  onClick={() => onNavigate('admin')}
                  style={{
                    width: '100%',
                    padding: '10px',
                    marginTop: '8px',
                    background: 'rgba(255, 68, 68, 0.1)',
                    border: '1px solid #ff4444',
                    color: '#ff4444',
                    borderRadius: '6px',
                    cursor: 'pointer',
                    fontWeight: '600',
                    transition: 'all 0.3s ease'
                  }}
                  className="trend-item"
                >
                  👑 Admin Dashboard
                </button>
              )}
            </div>
          )}

          {/* Request Moderation - Only for regular users */}
          {userData.role === 'user' && (
            <div className="trends-section">
              <h3 className="trends-title">Moderation</h3>
              {userData.moderation_requested ? (
                <div style={{
                  padding: '12px',
                  background: 'rgba(255, 170, 0, 0.1)',
                  border: '1px solid #ffaa00',
                  borderRadius: '6px',
                  color: '#ffaa00',
                  fontSize: '14px',
                  textAlign: 'center'
                }}>
                  ⏳ Moderation request pending approval
                </div>
              ) : (
                <button
                  onClick={async () => {
                    try {
                      await adminApi.requestModeration();
                      if (updateUserData) {
                        updateUserData({ ...userData, moderation_requested: true });
                      }
                      alert('Moderation request submitted! An admin will review it.');
                    } catch (err) {
                      alert('Failed to submit request: ' + err.message);
                    }
                  }}
                  style={{
                    width: '100%',
                    padding: '10px',
                    background: 'rgba(0, 255, 136, 0.1)',
                    border: '1px solid #00ff88',
                    color: '#00ff88',
                    borderRadius: '6px',
                    cursor: 'pointer',
                    fontWeight: '600',
                    transition: 'all 0.3s ease'
                  }}
                  className="trend-item"
                >
                  Request Moderation Access
                </button>
              )}
              <p style={{
                fontSize: '12px',
                color: '#666',
                marginTop: '8px',
                lineHeight: '1.4'
              }}>
                Want to help moderate this server? Request access and an admin will review your application.
              </p>
            </div>
          )}

          <div className="trends-section">
            <h3 className="trends-title">Coming Soon</h3>
            <ul className="features-list">
              <li>WebFinger Discovery - Sprint 2</li>
              <li>ActivityPub Federation - Sprint 2</li>
              <li>Instance Blocking - Sprint 2</li>
              <li>Reply Threading - Sprint 2</li>
              <li>E2E Encrypted DMs - Sprint 2</li>
              <li>Media Upload UI - Sprint 2</li>
              <li>Content Reporting - Sprint 2</li>
              <li>Hashtag Support - Sprint 3</li>
              <li>Trending Topics - Sprint 3</li>
              <li>Mobile PWA - Sprint 3</li>
            </ul>
          </div>
        </aside>
      </div>
    </div>
  );
}
