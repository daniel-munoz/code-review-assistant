/**
 * Sample TypeScript module for testing the code analyzer.
 *
 * This module contains various functions and classes to test
 * metrics extraction and anti-pattern detection.
 */

import { readFile } from 'fs';
import path from 'path';

// Type definitions
interface User {
  id: number;
  name: string;
  email: string;
}

type Status = 'active' | 'inactive' | 'pending';

// Simple function
function greet(name: string): string {
  return `Hello, ${name}!`;
}

// Function with multiple parameters
function calculateTotal(items: number[], taxRate: number, discount: number = 0): number {
  const subtotal = items.reduce((sum, item) => sum + item, 0);
  const tax = subtotal * taxRate;
  const total = subtotal + tax - discount;
  return total;
}

// Arrow function assigned to const
const processData = (data: Record<string, unknown>[], filterKey?: string): Record<string, unknown>[] => {
  const result: Record<string, unknown>[] = [];

  for (const item of data) {
    if (filterKey === undefined) {
      result.push(item);
    } else if (filterKey in item) {
      if (item[filterKey] !== null) {
        result.push(item);
      }
    }
  }

  return result;
};

// Function expression
const formatCurrency = function(amount: number, currency: string = 'USD'): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency }).format(amount);
};

// Async function
async function fetchUser(id: number): Promise<User | null> {
  try {
    const response = await fetch(`/api/users/${id}`);
    if (!response.ok) {
      return null;
    }
    return response.json();
  } catch (error) {
    console.error('Failed to fetch user:', error);
    return null;
  }
}

// Generator function
function* numberGenerator(max: number): Generator<number> {
  for (let i = 0; i < max; i++) {
    yield i;
  }
}

// Class with various method types
class DataProcessor {
  private name: string;
  private maxItems: number;
  private items: unknown[];

  constructor(name: string, maxItems: number = 100) {
    this.name = name;
    this.maxItems = maxItems;
    this.items = [];
  }

  addItem(item: unknown): boolean {
    if (this.items.length < this.maxItems) {
      this.items.push(item);
      return true;
    }
    return false;
  }

  getItems(): unknown[] {
    return [...this.items];
  }

  clear(): void {
    this.items = [];
  }

  // Arrow function as class field
  getName = (): string => {
    return this.name;
  };
}

// Class extending another class
class AdvancedProcessor extends DataProcessor {
  private enableLogging: boolean;
  private log: string[];

  constructor(name: string, maxItems: number = 100, enableLogging: boolean = false) {
    super(name, maxItems);
    this.enableLogging = enableLogging;
    this.log = [];
  }

  addItem(item: unknown): boolean {
    const result = super.addItem(item);
    if (this.enableLogging) {
      this.log.push(`Added item: ${result}`);
    }
    return result;
  }

  getLog(): string[] {
    return [...this.log];
  }
}

// Constants
const MAX_RETRIES = 3;
const DEFAULT_TIMEOUT = 30;
const PI = 3.14159;

// Main function
function main(): void {
  const processor = new DataProcessor('test');
  processor.addItem({ id: 1, value: 'test' });
  console.log(greet('World'));
}

export {
  greet,
  calculateTotal,
  processData,
  formatCurrency,
  fetchUser,
  numberGenerator,
  DataProcessor,
  AdvancedProcessor,
  main,
};
