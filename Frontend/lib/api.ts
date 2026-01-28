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
    const error = await response.json().catch(() => ({ error: 'Network error' }));
    throw new Error(error.error || `HTTP ${response.status}`);
  }
  return response.json();
}

// Auth API
export const authApi = {
  async register(data: {
    username: string;
    instance_domain: string;
    did: string;
    display_name: string;
    public_key: string;
    bio?: string;
    avatar_url?: string;
  }) {
    const response = await fetch(`${API_BASE}/auth/register`, {
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

  async getChallenge(did: string) {
    const response = await fetch(`${API_BASE}/auth/challenge`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ did })
    });
    return handleResponse<{ challenge: string; expires_at: number }>(response);
  },

  async verifyChallenge(data: {
    did: string;
    challenge: string;
    signature: string;
  }) {
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

  logout() {
    if (typeof window !== 'undefined') {
      localStorage.removeItem('jwt_token');
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
  async createPost(content: string, imageUrl?: string) {
    const response = await fetch(`${API_BASE}/posts`, {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify({ content, image_url: imageUrl })
    });
    return handleResponse<any>(response);
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
    const response = await fetch(`${API_BASE}/users/${userId}/follow`, {
      method: 'POST',
      headers: getAuthHeaders()
    });
    return handleResponse<any>(response);
  },

  async unfollowUser(userId: string) {
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

// Export all APIs
export const api = {
  auth: authApi,
  user: userApi,
  post: postApi,
  follow: followApi,
  interaction: interactionApi,
  health: healthApi
};

export default api;
