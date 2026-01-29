// Crypto utilities for DID generation and signing (browser-side)
// This runs in the browser using Web Crypto API

export interface KeyPair {
  publicKey: CryptoKey;
  privateKey: CryptoKey;
  publicKeyBase64: string;
  privateKeyBase64: string;
  did: string;
}

// Generate a new Ed25519 keypair for DID
export async function generateKeyPair(): Promise<KeyPair> {
  // Generate Ed25519 keypair (or fallback to RSA if not supported)
  const keyPair = await window.crypto.subtle.generateKey(
    {
      name: 'ECDSA',
      namedCurve: 'P-256', // Using P-256 as Ed25519 isn't universally supported in browsers
    },
    true, // extractable
    ['sign', 'verify']
  );

  // Export public key
  const publicKeyBuffer = await window.crypto.subtle.exportKey('spki', keyPair.publicKey);
  const publicKeyBase64 = bufferToBase64(publicKeyBuffer);

  // Export private key
  const privateKeyBuffer = await window.crypto.subtle.exportKey('pkcs8', keyPair.privateKey);
  const privateKeyBase64 = bufferToBase64(privateKeyBuffer);

  // Generate DID from public key
  const did = await generateDID(publicKeyBase64);

  return {
    publicKey: keyPair.publicKey,
    privateKey: keyPair.privateKey,
    publicKeyBase64,
    privateKeyBase64,
    did,
  };
}

// Generate DID from public key
async function generateDID(publicKeyBase64: string): Promise<string> {
  // Create a simple DID:key from the public key
  const hash = await hashString(publicKeyBase64);
  return `did:key:z6Mk${hash.substring(0, 40)}`;
}

// Sign a challenge with private key
export async function signChallenge(privateKey: CryptoKey, challenge: string): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(challenge);

  const signature = await window.crypto.subtle.sign(
    {
      name: 'ECDSA',
      hash: { name: 'SHA-256' },
    },
    privateKey,
    data
  );

  return bufferToBase64(signature);
}

// Import a private key from base64
export async function importPrivateKey(base64Key: string): Promise<CryptoKey> {
  const buffer = base64ToBuffer(base64Key);

  return await window.crypto.subtle.importKey(
    'pkcs8',
    buffer,
    {
      name: 'ECDSA',
      namedCurve: 'P-256',
    },
    true,
    ['sign']
  );
}

// Helper: Convert ArrayBuffer to Base64
function bufferToBase64(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary);
}

// Helper: Convert Base64 to ArrayBuffer
function base64ToBuffer(base64: string): ArrayBuffer {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer;
}

// Helper: Hash a string
async function hashString(str: string): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(str);
  const hashBuffer = await window.crypto.subtle.digest('SHA-256', data);
  return bufferToBase64(hashBuffer);
}

// Store keypair in localStorage
export function storeKeyPair(keyPair: KeyPair): void {
  localStorage.setItem('private_key', keyPair.privateKeyBase64);
  localStorage.setItem('public_key', keyPair.publicKeyBase64);
  localStorage.setItem('did', keyPair.did);
}

// Retrieve keypair from localStorage
export async function loadKeyPair(): Promise<KeyPair | null> {
  const privateKeyBase64 = localStorage.getItem('private_key');
  const publicKeyBase64 = localStorage.getItem('public_key');
  const did = localStorage.getItem('did');

  if (!privateKeyBase64 || !publicKeyBase64 || !did) {
    return null;
  }

  try {
    const privateKey = await importPrivateKey(privateKeyBase64);
    // We don't need to import public key for signing, but store for completeness
    return {
      privateKey,
      publicKey: null as any, // Not needed for signing
      publicKeyBase64,
      privateKeyBase64,
      did,
    };
  } catch (error) {
    console.error('Failed to load keypair:', error);
    return null;
  }
}

// Export recovery file
export function exportRecoveryFile(keyPair: KeyPair, username: string, server: string): void {
  const recoveryData = {
    did: keyPair.did,
    public_key: keyPair.publicKeyBase64,
    private_key: keyPair.privateKeyBase64,
    username,
    server,
    timestamp: new Date().toISOString(),
    warning: 'KEEP THIS FILE SECURE! Anyone with this file can access your account.',
  };

  const dataStr = JSON.stringify(recoveryData, null, 2);
  const dataBlob = new Blob([dataStr], { type: 'application/json' });
  const url = URL.createObjectURL(dataBlob);
  
  const link = document.createElement('a');
  link.href = url;
  link.download = `splitter-recovery-${username}-${Date.now()}.json`;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

// Clear stored keys (logout)
export function clearStoredKeys(): void {
  localStorage.removeItem('private_key');
  localStorage.removeItem('public_key');
  localStorage.removeItem('did');
  localStorage.removeItem('jwt_token');
}

// Alias for loadKeyPair (for backward compatibility)
export const getStoredKeyPair = loadKeyPair;

// Import recovery file
export async function importRecoveryFile(file: File): Promise<KeyPair> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    
    reader.onload = async (e) => {
      try {
        const content = e.target?.result as string;
        const recoveryData = JSON.parse(content);
        
        if (!recoveryData.private_key || !recoveryData.public_key || !recoveryData.did) {
          throw new Error('Invalid recovery file format');
        }
        
        const privateKey = await importPrivateKey(recoveryData.private_key);
        
        const keyPair: KeyPair = {
          privateKey,
          publicKey: null as any,
          publicKeyBase64: recoveryData.public_key,
          privateKeyBase64: recoveryData.private_key,
          did: recoveryData.did,
        };
        
        // Store the imported keys
        storeKeyPair(keyPair);
        
        resolve(keyPair);
      } catch (error) {
        reject(new Error('Failed to parse recovery file'));
      }
    };
    
    reader.onerror = () => reject(new Error('Failed to read file'));
    reader.readAsText(file);
  });
}
