'use client';

/**
 * Dynamic Confetti Component
 *
 * Code-split confetti animation with proper client-side handling
 * and performance optimizations.
 */

import { useEffect, useState, useCallback } from 'react';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { preloadHeavyComponents } from '@/lib/dynamic-imports';
import { Sparkles, PartyPopper } from 'lucide-react';

import type confetti from 'canvas-confetti';

type ConfettiFunction = typeof confetti;

interface ConfettiOptions {
  particleCount?: number;
  angle?: number;
  spread?: number;
  startVelocity?: number;
  decay?: number;
  gravity?: number;
  drift?: number;
  flat?: boolean;
  ticks?: number;
  origin?: { x: number; y: number };
  colors?: string[];
  shapes?: confetti.Shape[];
  scalar?: number;
  zIndex?: number;
}

export function DynamicConfetti() {
  const [isClient, setIsClient] = useState(false);
  const [confetti, setConfetti] = useState<ConfettiFunction | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  // Ensure we're on the client side
  useEffect(() => {
    setIsClient(true);
  }, []);

  // Lazy load confetti function
  const loadConfetti = useCallback(async () => {
    if (confetti) return confetti;

    setIsLoading(true);
    try {
      const confettiModule = await preloadHeavyComponents.confetti();
      const confettiFunc = confettiModule.default;
      setConfetti(confettiFunc);
      setIsLoading(false);
      return confettiFunc;
    } catch (error) {
      console.error('Failed to load confetti:', error);
      setIsLoading(false);
      return null;
    }
  }, [confetti]);

  // Trigger confetti with various effects
  const triggerConfetti = useCallback(
    async (type: 'basic' | 'burst' | 'rainbow' | 'school-pride' = 'basic') => {
      const confettiFunc = await loadConfetti();
      if (!confettiFunc) return;

      const commonOptions = {
        particleCount: 100,
        spread: 70,
        origin: { y: 0.6 },
      };

      switch (type) {
        case 'basic':
          confettiFunc({
            ...commonOptions,
            colors: ['#26ccff', '#a25afd', '#ff5722', '#ff9800', '#4caf50'],
          });
          break;

        case 'burst': {
          // Multiple bursts
          const count = 200;
          const defaults = {
            origin: { y: 0.7 },
          };

          const fire = (particleRatio: number, opts: ConfettiOptions) => {
            confettiFunc?.({
              ...defaults,
              ...opts,
              particleCount: Math.floor(count * particleRatio),
            });
          };

          fire(0.25, {
            spread: 26,
            startVelocity: 55,
          });
          fire(0.2, {
            spread: 60,
          });
          fire(0.35, {
            spread: 100,
            decay: 0.91,
            scalar: 0.8,
          });
          fire(0.1, {
            spread: 120,
            startVelocity: 25,
            decay: 0.92,
            scalar: 1.2,
          });
          fire(0.1, {
            spread: 120,
            startVelocity: 45,
          });
          break;
        }

        case 'rainbow':
          // Rainbow confetti
          confettiFunc({
            particleCount: 150,
            spread: 60,
            origin: { y: 0.6 },
            colors: [
              '#ff0000',
              '#ff8000',
              '#ffff00',
              '#80ff00',
              '#00ff00',
              '#00ff80',
              '#00ffff',
              '#0080ff',
              '#0000ff',
              '#8000ff',
              '#ff00ff',
              '#ff0080',
            ],
          });
          break;

        case 'school-pride': {
          // Custom colors
          const end = Date.now() + 3 * 1000; // 3 seconds
          const colors = ['#bb0000', '#ffffff'];

          (function frame() {
            confettiFunc({
              particleCount: 2,
              angle: 60,
              spread: 55,
              origin: { x: 0 },
              colors: colors,
            });
            confettiFunc({
              particleCount: 2,
              angle: 120,
              spread: 55,
              origin: { x: 1 },
              colors: colors,
            });

            if (Date.now() < end) {
              requestAnimationFrame(frame);
            }
          })();
          break;
        }
      }
    },
    [loadConfetti]
  );

  if (!isClient) {
    return null; // Don't render on server
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <PartyPopper className="w-5 h-5" />
          Celebration Animations
        </CardTitle>
        <CardDescription>
          Click the buttons to trigger different confetti animations
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-2 gap-4">
          <Button
            onClick={() => triggerConfetti('basic')}
            disabled={isLoading}
            variant="outline"
          >
            <Sparkles className="w-4 h-4 mr-2" />
            Basic Confetti
          </Button>

          <Button
            onClick={() => triggerConfetti('burst')}
            disabled={isLoading}
            variant="outline"
          >
            <PartyPopper className="w-4 h-4 mr-2" />
            Burst Effect
          </Button>

          <Button
            onClick={() => triggerConfetti('rainbow')}
            disabled={isLoading}
            variant="outline"
          >
            <Sparkles className="w-4 h-4 mr-2" />
            Rainbow Colors
          </Button>

          <Button
            onClick={() => triggerConfetti('school-pride')}
            disabled={isLoading}
            variant="outline"
          >
            <PartyPopper className="w-4 h-4 mr-2" />
            School Pride
          </Button>
        </div>

        {isLoading && (
          <div className="mt-4 text-center text-sm text-muted-foreground">
            Loading animation...
          </div>
        )}
      </CardContent>
    </Card>
  );
}

/**
 * Success confetti hook for use in other components
 */
export function useSuccessConfetti() {
  const [isClient, setIsClient] = useState(false);
  const [confetti, setConfetti] = useState<ConfettiFunction | null>(null);

  useEffect(() => {
    setIsClient(true);
  }, []);

  const celebrate = useCallback(async () => {
    if (!isClient) return;

    try {
      if (!confetti) {
        const confettiModule = await preloadHeavyComponents.confetti();
        const confettiFunc = confettiModule.default;
        setConfetti(confettiFunc);

        // Trigger celebration
        confettiFunc({
          particleCount: 100,
          spread: 70,
          origin: { y: 0.6 },
          colors: ['#26ccff', '#a25afd', '#ff5722', '#ff9800', '#4caf50'],
        });
      } else {
        confetti({
          particleCount: 100,
          spread: 70,
          origin: { y: 0.6 },
          colors: ['#26ccff', '#a25afd', '#ff5722', '#ff9800', '#4caf50'],
        });
      }
    } catch (error) {
      console.warn('Confetti animation failed:', error);
    }
  }, [isClient, confetti]);

  return { celebrate, isAvailable: isClient };
}

/**
 * Confetti trigger for server operations success
 */
export function SuccessConfetti({
  trigger,
  onComplete,
}: {
  trigger: boolean;
  onComplete?: () => void;
}) {
  const { celebrate } = useSuccessConfetti();

  useEffect(() => {
    if (trigger) {
      celebrate();
      onComplete?.();
    }
  }, [trigger, celebrate, onComplete]);

  return null; // This component doesn't render anything
}
