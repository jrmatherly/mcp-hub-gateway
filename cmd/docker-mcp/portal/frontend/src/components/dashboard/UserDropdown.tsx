'use client';

import { useState, useRef, useEffect, useCallback } from 'react';
import { useMsal } from '@azure/msal-react';
import Image from 'next/image';
import {
  User,
  LogOut,
  Settings,
  HelpCircle,
  ChevronDown,
  UserCircle,
} from 'lucide-react';
import { useRouter } from 'next/navigation';

export function UserDropdown() {
  const { instance, accounts } = useMsal();
  const [isOpen, setIsOpen] = useState(false);
  const [userPhoto, setUserPhoto] = useState<string | null>(null);
  const [photoLoading, setPhotoLoading] = useState(true);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const router = useRouter();

  // Get the current user account
  const account = accounts[0];

  // Fetch user photo from Microsoft Graph
  const fetchUserPhoto = useCallback(async () => {
    if (!account) {
      setPhotoLoading(false);
      return;
    }

    try {
      // Get access token for Microsoft Graph
      const tokenResponse = await instance.acquireTokenSilent({
        scopes: ['User.Read'],
        account: account,
      });

      // Fetch photo from Microsoft Graph
      const response = await fetch(
        'https://graph.microsoft.com/v1.0/me/photo/$value',
        {
          headers: {
            Authorization: `Bearer ${tokenResponse.accessToken}`,
          },
        }
      );

      if (response.ok) {
        const blob = await response.blob();
        const photoUrl = URL.createObjectURL(blob);
        setUserPhoto(photoUrl);
      }
    } catch (_error) {
      // Photo not available or error fetching - will fallback to initials
      // Using debug level log for non-critical photo fetch failure
    } finally {
      setPhotoLoading(false);
    }
  }, [account, instance]);

  // Fetch photo when component mounts or account changes
  useEffect(() => {
    fetchUserPhoto();
  }, [fetchUserPhoto]);

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
      }
    }

    document.addEventListener('mousedown', handleClickOutside);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, []);

  // Cleanup photo URL on unmount
  useEffect(() => {
    return () => {
      if (userPhoto) {
        URL.revokeObjectURL(userPhoto);
      }
    };
  }, [userPhoto]);

  const handleLogout = async () => {
    try {
      await instance.logoutPopup({
        postLogoutRedirectUri: '/',
        mainWindowRedirectUri: '/',
      });
    } catch (error) {
      console.error('Logout failed:', error);
      // Fallback to redirect logout
      await instance.logoutRedirect({
        postLogoutRedirectUri: '/',
      });
    }
  };

  const handleSettings = () => {
    router.push('/settings');
    setIsOpen(false);
  };

  const handleHelp = () => {
    router.push('/help');
    setIsOpen(false);
  };

  if (!account) {
    return (
      <button
        onClick={() => router.push('/')}
        className="flex items-center gap-2 px-3 py-2 text-sm rounded-md hover:bg-muted transition-colors"
      >
        <UserCircle className="h-5 w-5" />
        <span>Sign In</span>
      </button>
    );
  }

  // Extract user initials for avatar
  const getInitials = (name: string | undefined) => {
    if (!name) return account.username?.substring(0, 2).toUpperCase() || 'U';
    return name
      .split(' ')
      .map(n => n[0])
      .join('')
      .toUpperCase()
      .substring(0, 2);
  };

  const userInitials = getInitials(account.name || undefined);
  const displayName = account.name || account.username || 'User';
  const email = account.username || '';

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-2 px-3 py-1.5 rounded-md hover:bg-muted transition-colors"
        aria-expanded={isOpen}
        aria-haspopup="true"
      >
        {/* User Avatar */}
        <div className="relative w-8 h-8">
          {photoLoading ? (
            // Loading state
            <div className="flex items-center justify-center w-8 h-8 rounded-full bg-muted animate-pulse" />
          ) : userPhoto ? (
            // User photo
            <Image
              src={userPhoto}
              alt={displayName}
              width={32}
              height={32}
              className="rounded-full object-cover border border-border"
              onError={() => {
                // If image fails to load, clear it to show initials
                setUserPhoto(null);
              }}
              unoptimized // Microsoft Graph photos are already optimized
            />
          ) : (
            // Fallback to initials
            <div className="flex items-center justify-center w-8 h-8 rounded-full bg-primary text-primary-foreground font-semibold text-sm">
              {userInitials}
            </div>
          )}
        </div>

        {/* User Name (hidden on mobile) */}
        <span className="hidden md:block text-sm font-medium">
          {displayName.split(' ')[0]}
        </span>

        {/* Dropdown Chevron */}
        <ChevronDown
          className={`h-4 w-4 transition-transform ${isOpen ? 'rotate-180' : ''}`}
        />
      </button>

      {/* Dropdown Menu */}
      {isOpen && (
        <div className="absolute right-0 mt-2 w-64 rounded-md border bg-popover p-1 shadow-lg z-50">
          {/* User Info Header */}
          <div className="px-3 py-2 border-b mb-1">
            <div className="flex items-center gap-3">
              <div className="relative w-10 h-10">
                {userPhoto ? (
                  <Image
                    src={userPhoto}
                    alt={displayName}
                    width={40}
                    height={40}
                    className="rounded-full object-cover border border-border"
                    onError={() => setUserPhoto(null)}
                    unoptimized // Microsoft Graph photos are already optimized
                  />
                ) : (
                  <div className="flex items-center justify-center w-10 h-10 rounded-full bg-primary text-primary-foreground font-semibold">
                    {userInitials}
                  </div>
                )}
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium truncate">{displayName}</p>
                <p className="text-xs text-muted-foreground truncate">
                  {email}
                </p>
              </div>
            </div>
          </div>

          {/* Menu Items */}
          <div className="py-1">
            <button
              onClick={() => {
                router.push('/profile');
                setIsOpen(false);
              }}
              className="flex items-center gap-3 w-full px-3 py-2 text-sm rounded-sm hover:bg-muted transition-colors"
            >
              <User className="h-4 w-4" />
              <span>Profile</span>
            </button>

            <button
              onClick={handleSettings}
              className="flex items-center gap-3 w-full px-3 py-2 text-sm rounded-sm hover:bg-muted transition-colors"
            >
              <Settings className="h-4 w-4" />
              <span>Settings</span>
            </button>

            <button
              onClick={handleHelp}
              className="flex items-center gap-3 w-full px-3 py-2 text-sm rounded-sm hover:bg-muted transition-colors"
            >
              <HelpCircle className="h-4 w-4" />
              <span>Help & Support</span>
            </button>

            {/* Divider */}
            <div className="my-1 border-t" />

            {/* Logout Button */}
            <button
              onClick={handleLogout}
              className="flex items-center gap-3 w-full px-3 py-2 text-sm rounded-sm hover:bg-destructive hover:text-destructive-foreground transition-colors"
            >
              <LogOut className="h-4 w-4" />
              <span>Sign Out</span>
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
