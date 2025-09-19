'use client';

import { AlertCircle, Eye, EyeOff, Loader2, LogIn } from 'lucide-react';
import Link from 'next/link';
import { useState, useEffect } from 'react';
import { useMsal } from '@azure/msal-react';
import { useRouter } from 'next/navigation';
import { authLogger } from '@/lib/logger';
import { Button } from '@/components/ui/button';
import { loginRequest } from '@/config/msal.config';

interface LoginFormData {
  email: string;
  password: string;
  rememberMe: boolean;
}

interface FormErrors {
  email?: string;
  password?: string;
  general?: string;
}

export default function LoginPage() {
  const { instance } = useMsal();
  const router = useRouter();

  const [formData, setFormData] = useState<LoginFormData>({
    email: '',
    password: '',
    rememberMe: false,
  });

  const [errors, setErrors] = useState<FormErrors>({});
  const [showPassword, setShowPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  // Check if already authenticated and redirect
  useEffect(() => {
    const accounts = instance.getAllAccounts();
    if (accounts.length > 0) {
      authLogger.info('User already authenticated, redirecting to dashboard');
      router.push('/dashboard');
    }
  }, [instance, router]);

  const validateForm = (): FormErrors => {
    const newErrors: FormErrors = {};

    if (!formData.email) {
      newErrors.email = 'Email is required';
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = 'Please enter a valid email address';
    }

    if (!formData.password) {
      newErrors.password = 'Password is required';
    } else if (formData.password.length < 6) {
      newErrors.password = 'Password must be at least 6 characters';
    }

    return newErrors;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const newErrors = validateForm();
    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors);
      return;
    }

    setIsLoading(true);
    setErrors({});

    try {
      // TODO: Implement actual login logic with Azure AD MSAL
      authLogger.debug('Login attempt started', {
        email: formData.email,
        rememberMe: formData.rememberMe,
      });

      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 2000));

      // For now, just log the attempt
      authLogger.info('Login successful (simulated)');

      // TODO: Redirect to dashboard
      window.location.href = '/dashboard';
    } catch (loginError) {
      authLogger.error('Login failed', loginError);
      setErrors({
        general: 'Login failed. Please check your credentials and try again.',
      });
    } finally {
      setIsLoading(false);
    }
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value, type, checked } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value,
    }));

    // Clear error when user starts typing
    if (errors[name as keyof FormErrors]) {
      setErrors(prev => ({
        ...prev,
        [name]: undefined,
      }));
    }
  };

  const handleAzureLogin = async () => {
    setIsLoading(true);
    try {
      authLogger.debug('Azure AD login initiated');

      // Use MSAL redirect login for better compatibility
      // Note: If Azure AD app is registered as "Web" instead of "SPA",
      // you need to change it to "Single-page application" in Azure Portal
      const loginResponse = await instance.loginPopup(loginRequest);

      authLogger.info('Azure AD login successful', {
        username: loginResponse.account?.username,
        homeAccountId: loginResponse.account?.homeAccountId,
      });

      // The router.push will trigger the AuthGuard to check authentication
      router.push('/dashboard');
    } catch (azureError) {
      authLogger.error('Azure AD login failed', azureError);

      // Handle specific MSAL errors
      let errorMessage = 'Azure AD login failed. Please try again.';

      // Type guard for MSAL error
      const msalError = azureError as {
        errorCode?: string;
        errorMessage?: string;
      };

      if (msalError.errorCode === 'user_cancelled') {
        errorMessage = 'Login was cancelled.';
      } else if (msalError.errorCode === 'consent_required') {
        errorMessage =
          'Additional permissions are required. Please contact your administrator.';
      } else if (msalError.errorCode === 'interaction_in_progress') {
        errorMessage = 'Login already in progress. Please wait.';
      } else if (msalError.errorCode === 'invalid_request') {
        // Check for the specific SPA configuration error
        if (msalError.errorMessage?.includes('AADSTS9002326')) {
          errorMessage =
            'Azure AD configuration error: The app must be registered as a "Single-page application" in Azure Portal, not as a "Web" application. Please contact your administrator to update the app registration.';
        }
      }

      setErrors({
        general: errorMessage,
      });
      setIsLoading(false);
    }
  };

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="text-center">
        <h1 className="text-3xl font-bold text-foreground mb-2">Sign In</h1>
        <p className="text-muted-foreground">
          Access your MCP Portal dashboard
        </p>
      </div>

      {/* Azure AD Login */}
      <div>
        <Button
          onClick={handleAzureLogin}
          disabled={isLoading}
          size="lg"
          className="w-full"
        >
          {isLoading ? (
            <Loader2 className="h-5 w-5 animate-spin mr-2" />
          ) : (
            <LogIn className="h-5 w-5 mr-2" />
          )}
          Sign in with Microsoft
        </Button>

        <div className="relative my-6">
          <div className="absolute inset-0 flex items-center">
            <span className="w-full border-t" />
          </div>
          <div className="relative flex justify-center text-xs uppercase">
            <span className="bg-background px-2 text-muted-foreground">
              Or continue with email
            </span>
          </div>
        </div>
      </div>

      {/* Login Form */}
      <form onSubmit={handleSubmit} className="space-y-4">
        {/* General Error */}
        {errors.general && (
          <div className="flex items-center gap-2 p-3 text-sm text-error-700 bg-error-50 border border-error-200 rounded-md dark:text-error-300 dark:bg-error-950 dark:border-error-800">
            <AlertCircle className="h-4 w-4 flex-shrink-0" />
            <span>{errors.general}</span>
          </div>
        )}

        {/* Email Field */}
        <div>
          <label
            htmlFor="email"
            className="block text-sm font-medium text-foreground mb-2"
          >
            Email Address
          </label>
          <input
            type="email"
            id="email"
            name="email"
            value={formData.email}
            onChange={handleInputChange}
            className={`input ${errors.email ? 'border-error-500 focus-visible:ring-error-500' : ''}`}
            placeholder="Enter your email"
            disabled={isLoading}
            autoComplete="email"
          />
          {errors.email && (
            <p className="mt-1 text-sm text-error-600 dark:text-error-400">
              {errors.email}
            </p>
          )}
        </div>

        {/* Password Field */}
        <div>
          <label
            htmlFor="password"
            className="block text-sm font-medium text-foreground mb-2"
          >
            Password
          </label>
          <div className="relative">
            <input
              type={showPassword ? 'text' : 'password'}
              id="password"
              name="password"
              value={formData.password}
              onChange={handleInputChange}
              className={`input pr-10 ${errors.password ? 'border-error-500 focus-visible:ring-error-500' : ''}`}
              placeholder="Enter your password"
              disabled={isLoading}
              autoComplete="current-password"
            />
            <button
              type="button"
              onClick={() => setShowPassword(!showPassword)}
              className="absolute inset-y-0 right-0 flex items-center pr-3 text-muted-foreground hover:text-foreground transition-colors"
              disabled={isLoading}
            >
              {showPassword ? (
                <EyeOff className="h-4 w-4" />
              ) : (
                <Eye className="h-4 w-4" />
              )}
            </button>
          </div>
          {errors.password && (
            <p className="mt-1 text-sm text-error-600 dark:text-error-400">
              {errors.password}
            </p>
          )}
        </div>

        {/* Remember Me & Forgot Password */}
        <div className="flex items-center justify-between">
          <label className="flex items-center gap-2 text-sm text-foreground">
            <input
              type="checkbox"
              name="rememberMe"
              checked={formData.rememberMe}
              onChange={handleInputChange}
              className="h-4 w-4 text-primary border rounded focus:ring-primary focus:ring-2"
              disabled={isLoading}
            />
            Remember me
          </label>

          <Link
            href="/auth/forgot-password"
            className="text-sm text-primary hover:underline"
          >
            Forgot password?
          </Link>
        </div>

        {/* Submit Button */}
        <Button type="submit" disabled={isLoading} size="lg" className="w-full">
          {isLoading ? (
            <>
              <Loader2 className="h-5 w-5 animate-spin mr-2" />
              Signing in...
            </>
          ) : (
            <>
              <LogIn className="h-5 w-5 mr-2" />
              Sign In
            </>
          )}
        </Button>
      </form>

      {/* Footer */}
      <div className="text-center text-sm text-muted-foreground">
        Don't have an account?{' '}
        <Link
          href="/auth/register"
          className="text-primary hover:underline font-medium"
        >
          Contact your administrator
        </Link>
      </div>
    </div>
  );
}
