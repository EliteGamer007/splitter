// API Service Layer for Splitter Frontend
// Connects to Go backend at http://localhost:8000/api/v1

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8000/api/v1';

// Helper to get auth headers
const getAuthHeaders = (): HeadersInit => {
  const token = typeof window !== 'undefined' ? localStorage.getItem('jwt_token') : null;
  return {
    'Content-Type': 'application/json',
    ...(token && { Authorization: `Bearer ${token}` })
  };
};

// Helper to handle API responses
async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const errorText = await response.text();
    console.error('API Error Response:', {
      status: response.status,
      statusText: response.statusText,
      url: response.url,
      body: errorText
    });

    try {
      const error = JSON.parse(errorText);
      throw new Error(error.error || `HTTP ${response.status}`);
    } catch (parseError) {
      throw new Error(errorText || `HTTP ${response.status}`);
    }
  }
  return response.json();
}

// Auth API
export const authApi = {
  async register(data: {
    username: string;
    email: string;
    password: string;
    display_name?: string;
  }) {
    const response = await fetch(`${API_BASE}/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    });
    const result = await handleResponse<{ user: any; token: string }>(response);
    if (result.token) {
      localStorage.setItem('jwt_token', result.token);
      if (result.user) {
        localStorage.setItem('user', JSON.stringify(result.user));
      }
    }
    return result;
  },

  async login(data: { username: string; password: string }) {
    const response = await fetch(`${API_BASE}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    });
    const result = await handleResponse<{ user: any; token: string }>(response);
    if (result.token) {
      localStorage.setItem('jwt_token', result.token);
      if (result.user) {
        localStorage.setItem('user', JSON.stringify(result.user));
      }
    }
    return result;
  },

  async getChallenge(did: string) {
    const response = await fetch(`${API_BASE}/auth/challenge`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ did })
    });
    return handleResponse<{ challenge: string; expires_at: number }>(response);
  },

  async verifyChallenge(data: { did: string; challenge: string; signature: string }) {
    const response = await fetch(`${API_BASE}/auth/verify`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    });
    const result = await handleResponse<{ user: any; token: string }>(response);
    if (result.token) {
      localStorage.setItem('jwt_token', result.token);
    }
    return result;
  },

  isLoggedIn() {
    if (typeof window === 'undefined') return false;
    return !!localStorage.getItem('jwt_token');
  },

  getCurrentUserFromStorage() {
    if (typeof window === 'undefined') return null;
    const user = localStorage.getItem('user');
    return user ? JSON.parse(user) : null;
  },

  logout() {
    if (typeof window !== 'undefined') {
      localStorage.removeItem('jwt_token');
      localStorage.removeItem('user');
      localStorage.removeItem('private_key');
      localStorage.removeItem('did');
    }
  }
};

// User API
export const userApi = {
  async getCurrentUser() {
    const response = await fetch(`${API_BASE}/users/me`, {
      headers: getAuthHeaders()
    });
    return handleResponse<any>(response);
  },

  async getUserProfile(id: string) {
    const response = await fetch(`${API_BASE}/users/${id}`);
    return handleResponse<any>(response);
  },

  async getUserByDID(did: string) {
    const response = await fetch(`${API_BASE}/users/did?did=${encodeURIComponent(did)}`);
    return handleResponse<any>(response);
  },

  async updateProfile(data: {
    display_name?: string;
    bio?: string;
    avatar_url?: string;
    default_visibility?: string;
    message_privacy?: string;
    account_locked?: boolean;
  }) {
    const response = await fetch(`${API_BASE}/users/me`, {
      method: 'PUT',
      headers: getAuthHeaders(),
      body: JSON.stringify(data)
    });
    return handleResponse<any>(response);
  },

  async deleteAccount() {
    const response = await fetch(`${API_BASE}/users/me`, {
      method: 'DELETE',
      headers: getAuthHeaders()
    });
    return handleResponse<{ message: string }>(response);
  }
};

// Post API
export const postApi = {
  async createPost(content: string, visibility: string = 'public', file?: File) {
    const fd = new FormData();

    // Always send content (even empty string)
    fd.append('content', content || '');

    fd.append('visibility', visibility);

    if (file) {
      fd.append('file', file);
    }

    const token = typeof window !== 'undefined' ? localStorage.getItem('jwt_token') : null;
    const headers: HeadersInit = {};
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    return fetch(`${API_BASE}/posts`, {
      method: 'POST',
      headers: headers, // IMPORTANT: DO NOT set Content-Type manually
      body: fd
    }).then(r => handleResponse<any>(r));
  },

  async getPost(id: string) {
    const response = await fetch(`${API_BASE}/posts/${id}`);
    return handleResponse<any>(response);
  },

  async getUserPosts(userId: string, limit = 20, offset = 0) {
    const response = await fetch(
      `${API_BASE}/posts/user/${userId}?limit=${limit}&offset=${offset}`
    );
    return handleResponse<any[]>(response);
  },

  async getFeed(limit = 20, offset = 0) {
    const response = await fetch(
      `${API_BASE}/posts/feed?limit=${limit}&offset=${offset}`,
      { headers: getAuthHeaders() }
    );
    return handleResponse<any[]>(response);
  },

  async getPublicFeed(limit = 20, offset = 0, localOnly = false) {
    const response = await fetch(
      `${API_BASE}/posts/public?limit=${limit}&offset=${offset}&local_only=${localOnly}`
    );
    return handleResponse<any[]>(response);
  },

  async updatePost(id: string, content: string, imageUrl?: string) {
    const response = await fetch(`${API_BASE}/posts/${id}`, {
      method: 'PUT',
      headers: getAuthHeaders(),
      body: JSON.stringify({ content, image_url: imageUrl })
    });
    return handleResponse<any>(response);
  },

  async deletePost(id: string) {
    const response = await fetch(`${API_BASE}/posts/${id}`, {
      method: 'DELETE',
      headers: getAuthHeaders()
    });
    return handleResponse<{ message: string }>(response);
  }
};

// Follow API
export const followApi = {
  async followUser(userId: string) {
    console.log('followApi.followUser called with userId:', userId);
    console.log('API endpoint:', `${API_BASE}/users/${userId}/follow`);
    const headers = getAuthHeaders();
    console.log('Auth headers:', headers);

    const response = await fetch(`${API_BASE}/users/${userId}/follow`, {
      method: 'POST',
      headers: headers
    });

    console.log('Follow response status:', response.status);
    const result = await handleResponse<any>(response);
    console.log('Follow result:', result);
    return result;
  },

  async unfollowUser(userId: string) {
    console.log('followApi.unfollowUser called with userId:', userId);
    const response = await fetch(`${API_BASE}/users/${userId}/follow`, {
      method: 'DELETE',
      headers: getAuthHeaders()
    });
    return handleResponse<{ message: string }>(response);
  },

  async getFollowers(userId: string, limit = 50, offset = 0) {
    const response = await fetch(
      `${API_BASE}/users/${userId}/followers?limit=${limit}&offset=${offset}`
    );
    return handleResponse<any[]>(response);
  },

  async getFollowing(userId: string, limit = 50, offset = 0) {
    const response = await fetch(
      `${API_BASE}/users/${userId}/following?limit=${limit}&offset=${offset}`
    );
    return handleResponse<any[]>(response);
  },

  async getFollowStats(userId: string) {
    const response = await fetch(`${API_BASE}/users/${userId}/stats`);
    return handleResponse<{ followers: number; following: number }>(response);
  }
};

// Interaction API
export const interactionApi = {
  async likePost(postId: string) {
    const response = await fetch(`${API_BASE}/posts/${postId}/like`, {
      method: 'POST',
      headers: getAuthHeaders()
    });
    return handleResponse<{ message: string }>(response);
  },

  async unlikePost(postId: string) {
    const response = await fetch(`${API_BASE}/posts/${postId}/like`, {
      method: 'DELETE',
      headers: getAuthHeaders()
    });
    return handleResponse<{ message: string }>(response);
  },

  async repostPost(postId: string) {
    const response = await fetch(`${API_BASE}/posts/${postId}/repost`, {
      method: 'POST',
      headers: getAuthHeaders()
    });
    return handleResponse<{ message: string }>(response);
  },

  async unrepostPost(postId: string) {
    const response = await fetch(`${API_BASE}/posts/${postId}/repost`, {
      method: 'DELETE',
      headers: getAuthHeaders()
    });
    return handleResponse<{ message: string }>(response);
  },

  async bookmarkPost(postId: string) {
    const response = await fetch(`${API_BASE}/posts/${postId}/bookmark`, {
      method: 'POST',
      headers: getAuthHeaders()
    });
    return handleResponse<{ message: string }>(response);
  },

  async unbookmarkPost(postId: string) {
    const response = await fetch(`${API_BASE}/posts/${postId}/bookmark`, {
      method: 'DELETE',
      headers: getAuthHeaders()
    });
    return handleResponse<{ message: string }>(response);
  },

  async getBookmarks() {
    const response = await fetch(`${API_BASE}/users/me/bookmarks`, {
      headers: getAuthHeaders()
    });
    return handleResponse<any[]>(response);
  }
};

// Health check
export const healthApi = {
  async check() {
    const response = await fetch(`${API_BASE}/health`);
    return handleResponse<{ status: string }>(response);
  }
};

// Search API
export const searchApi = {
  async searchUsers(query: string, limit = 20, offset = 0) {
    const response = await fetch(
      `${API_BASE}/users/search?q=${encodeURIComponent(query)}&limit=${limit}&offset=${offset}`,
      { headers: getAuthHeaders() }
    );
    return handleResponse<{ users: any[] }>(response);
  }
};

// Message API
export const messageApi = {
  async getThreads() {
    const response = await fetch(`${API_BASE}/messages/threads`, {
      headers: getAuthHeaders()
    });
    return handleResponse<{ threads: any[] }>(response);
  },

  async getMessages(threadId: string, limit = 50, offset = 0) {
    const response = await fetch(
      `${API_BASE}/messages/threads/${threadId}?limit=${limit}&offset=${offset}`,
      { headers: getAuthHeaders() }
    );
    return handleResponse<{ messages: any[], thread: any }>(response);
  },

  async sendMessage(recipientId: string, content: string) {
    const response = await fetch(`${API_BASE}/messages/send`, {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify({ recipient_id: recipientId, content })
    });
    return handleResponse<{ message: any, thread: any, recipient: any }>(response);
  },

  async startConversation(userId: string) {
    const response = await fetch(`${API_BASE}/messages/conversation/${userId}`, {
      method: 'POST',
      headers: getAuthHeaders()
    });
    return handleResponse<{ thread: any }>(response);
  },

  async markAsRead(threadId: string) {
    const response = await fetch(`${API_BASE}/messages/threads/${threadId}/read`, {
      method: 'POST',
      headers: getAuthHeaders()
    });
    return handleResponse<{ message: string }>(response);
  }
};

// Admin API
export const adminApi = {
  async getAllUsers(limit = 50, offset = 0) {
    const response = await fetch(
      `${API_BASE}/admin/users?limit=${limit}&offset=${offset}`,
      { headers: getAuthHeaders() }
    );
    return handleResponse<{ users: any[], total: number }>(response);
  },

  async getModerationRequests() {
    const response = await fetch(`${API_BASE}/admin/moderation-requests`, {
      headers: getAuthHeaders()
    });
    return handleResponse<{ requests: any[] }>(response);
  },

  async approveModerationRequest(userId: string) {
    const response = await fetch(`${API_BASE}/admin/moderation-requests/${userId}/approve`, {
      method: 'POST',
      headers: getAuthHeaders()
    });
    return handleResponse<{ message: string }>(response);
  },

  async rejectModerationRequest(userId: string) {
    const response = await fetch(`${API_BASE}/admin/moderation-requests/${userId}/reject`, {
      method: 'POST',
      headers: getAuthHeaders()
    });
    return handleResponse<{ message: string }>(response);
  },

  async updateUserRole(userId: string, role: string) {
    const response = await fetch(`${API_BASE}/admin/users/${userId}/role`, {
      method: 'PUT',
      headers: getAuthHeaders(),
      body: JSON.stringify({ role })
    });
    return handleResponse<{ message: string }>(response);
  },

  async suspendUser(userId: string) {
    const response = await fetch(`${API_BASE}/admin/users/${userId}/suspend`, {
      method: 'POST',
      headers: getAuthHeaders()
    });
    return handleResponse<{ message: string }>(response);
  },

  async unsuspendUser(userId: string) {
    const response = await fetch(`${API_BASE}/admin/users/${userId}/unsuspend`, {
      method: 'POST',
      headers: getAuthHeaders()
    });
    return handleResponse<{ message: string }>(response);
  },

  async getSuspendedUsers(limit = 50, offset = 0) {
    const response = await fetch(
      `${API_BASE}/admin/users/suspended?limit=${limit}&offset=${offset}`,
      { headers: getAuthHeaders() }
    );
    return handleResponse<{ users: any[] }>(response);
  },

  async requestModeration() {
    const response = await fetch(`${API_BASE}/users/me/request-moderation`, {
      method: 'POST',
      headers: getAuthHeaders()
    });
    return handleResponse<{ message: string }>(response);
  },

  // Content moderation queue
  async getModerationQueue() {
    const response = await fetch(`${API_BASE}/admin/moderation-queue`, {
      headers: getAuthHeaders()
    });
    return handleResponse<{ items: any[] }>(response);
  },

  async approveContent(reportId: string) {
    const response = await fetch(`${API_BASE}/admin/moderation-queue/${reportId}/approve`, {
      method: 'POST',
      headers: getAuthHeaders()
    });
    return handleResponse<{ message: string }>(response);
  },

  async removeContent(reportId: string) {
    const response = await fetch(`${API_BASE}/admin/moderation-queue/${reportId}/remove`, {
      method: 'POST',
      headers: getAuthHeaders()
    });
    return handleResponse<{ message: string }>(response);
  },

  async warnUser(userId: string, reason: string) {
    const response = await fetch(`${API_BASE}/admin/users/${userId}/warn`, {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify({ reason })
    });
    return handleResponse<{ message: string }>(response);
  },

  async blockDomain(domain: string) {
    const response = await fetch(`${API_BASE}/admin/domains/block`, {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify({ domain })
    });
    return handleResponse<{ message: string }>(response);
  },

  // Get admin action history (ban/unban log)
  async getAdminActions(limit = 50, offset = 0) {
    const response = await fetch(
      `${API_BASE}/admin/actions?limit=${limit}&offset=${offset}`,
      { headers: getAuthHeaders() }
    );
    // Fallback to empty array if endpoint not implemented yet
    try {
      return handleResponse<{ actions: any[] }>(response);
    } catch (err) {
      console.log('Admin actions endpoint not yet implemented, returning empty');
      return { actions: [] };
    }
  },

  // Suspend user with reason
  async suspendUserWithReason(userId: string, reason: string) {
    const response = await fetch(`${API_BASE}/admin/users/${userId}/suspend`, {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify({ reason })
    });
    return handleResponse<{ message: string }>(response);
  }
};

// Export all APIs
export const api = {
  auth: authApi,
  user: userApi,
  post: postApi,
  follow: followApi,
  interaction: interactionApi,
  health: healthApi,
  search: searchApi,
  message: messageApi,
  admin: adminApi
};

export default api;
