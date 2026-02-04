/**
 * Sample JSX file for testing React component parsing.
 */

import React, { useState, useEffect, useCallback } from 'react';
import PropTypes from 'prop-types';

// Simple functional component
function Greeting({ name }) {
  return <h1>Hello, {name}!</h1>;
}

Greeting.propTypes = {
  name: PropTypes.string.isRequired,
};

// Component with hooks
function Counter({ initialValue = 0 }) {
  const [count, setCount] = useState(initialValue);

  const increment = useCallback(() => {
    setCount((prev) => prev + 1);
  }, []);

  const decrement = useCallback(() => {
    setCount((prev) => prev - 1);
  }, []);

  return (
    <div className="counter">
      <button onClick={decrement}>-</button>
      <span>{count}</span>
      <button onClick={increment}>+</button>
    </div>
  );
}

Counter.propTypes = {
  initialValue: PropTypes.number,
};

// Arrow function component
const UserCard = ({ user, onSelect }) => {
  const handleClick = () => {
    if (onSelect) {
      onSelect(user);
    }
  };

  return (
    <div className="user-card" onClick={handleClick}>
      <img src={user.avatar} alt={user.name} />
      <h3>{user.name}</h3>
      <p>{user.email}</p>
    </div>
  );
};

UserCard.propTypes = {
  user: PropTypes.shape({
    id: PropTypes.number.isRequired,
    name: PropTypes.string.isRequired,
    email: PropTypes.string.isRequired,
    avatar: PropTypes.string,
  }).isRequired,
  onSelect: PropTypes.func,
};

// Component with useEffect
function DataFetcher({ url, render }) {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    let cancelled = false;

    async function fetchData() {
      try {
        setLoading(true);
        const response = await fetch(url);
        if (!response.ok) {
          throw new Error('Failed to fetch');
        }
        const result = await response.json();
        if (!cancelled) {
          setData(result);
          setError(null);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err.message);
          setData(null);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    fetchData();

    return () => {
      cancelled = true;
    };
  }, [url]);

  if (loading) {
    return <div>Loading...</div>;
  }

  if (error) {
    return <div>Error: {error}</div>;
  }

  return render(data);
}

DataFetcher.propTypes = {
  url: PropTypes.string.isRequired,
  render: PropTypes.func.isRequired,
};

// Class component
class TodoList extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      todos: props.initialTodos || [],
      newTodo: '',
    };
  }

  handleInputChange = (e) => {
    this.setState({ newTodo: e.target.value });
  };

  handleAddTodo = () => {
    const { newTodo, todos } = this.state;
    if (newTodo.trim()) {
      this.setState({
        todos: [...todos, { id: Date.now(), text: newTodo, completed: false }],
        newTodo: '',
      });
    }
  };

  handleToggle = (id) => {
    this.setState((prevState) => ({
      todos: prevState.todos.map((todo) =>
        todo.id === id ? { ...todo, completed: !todo.completed } : todo
      ),
    }));
  };

  handleDelete = (id) => {
    this.setState((prevState) => ({
      todos: prevState.todos.filter((todo) => todo.id !== id),
    }));
  };

  render() {
    const { todos, newTodo } = this.state;

    return (
      <div className="todo-list">
        <div className="add-todo">
          <input
            type="text"
            value={newTodo}
            onChange={this.handleInputChange}
            placeholder="Add a todo"
          />
          <button onClick={this.handleAddTodo}>Add</button>
        </div>
        <ul>
          {todos.map((todo) => (
            <li
              key={todo.id}
              className={todo.completed ? 'completed' : ''}
              onClick={() => this.handleToggle(todo.id)}
            >
              {todo.text}
              <button onClick={() => this.handleDelete(todo.id)}>Delete</button>
            </li>
          ))}
        </ul>
      </div>
    );
  }
}

TodoList.propTypes = {
  initialTodos: PropTypes.arrayOf(
    PropTypes.shape({
      id: PropTypes.number.isRequired,
      text: PropTypes.string.isRequired,
      completed: PropTypes.bool.isRequired,
    })
  ),
};

// Higher-order component
function withLoading(WrappedComponent) {
  return function WithLoadingComponent({ isLoading, ...props }) {
    if (isLoading) {
      return <div className="loading-spinner">Loading...</div>;
    }
    return <WrappedComponent {...props} />;
  };
}

const UserCardWithLoading = withLoading(UserCard);

// Custom hook
function useLocalStorage(key, initialValue) {
  const [storedValue, setStoredValue] = useState(() => {
    try {
      const item = window.localStorage.getItem(key);
      return item ? JSON.parse(item) : initialValue;
    } catch (error) {
      console.error(error);
      return initialValue;
    }
  });

  const setValue = (value) => {
    try {
      const valueToStore = value instanceof Function ? value(storedValue) : value;
      setStoredValue(valueToStore);
      window.localStorage.setItem(key, JSON.stringify(valueToStore));
    } catch (error) {
      console.error(error);
    }
  };

  return [storedValue, setValue];
}

export {
  Greeting,
  Counter,
  UserCard,
  DataFetcher,
  TodoList,
  withLoading,
  UserCardWithLoading,
  useLocalStorage,
};
