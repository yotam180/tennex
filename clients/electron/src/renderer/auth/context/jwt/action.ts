import axios, { endpoints } from 'src/lib/axios';

import { setSession } from './utils';
import { JWT_STORAGE_KEY } from './constant';

// ----------------------------------------------------------------------

export type SignInParams = {
  username: string; // Go backend uses 'username' field
  password: string;
};

export type SignUpParams = {
  username: string;
  email: string;
  password: string;
  first_name: string;
  last_name: string;
};

/** **************************************
 * Sign in
 *************************************** */
export const signInWithPassword = async ({ username, password }: SignInParams): Promise<void> => {
  try {
    const params = { username, password };

    const res = await axios.post(endpoints.auth.signIn, params);

    const { token } = res.data; // Go backend returns 'token' field

    if (!token) {
      throw new Error('Access token not found in response');
    }

    setSession(token);
  } catch (error) {
    console.error('Error during sign in:', error);
    throw error;
  }
};

/** **************************************
 * Sign up
 *************************************** */
export const signUp = async ({
  username,
  email,
  password,
  first_name,
  last_name,
}: SignUpParams): Promise<void> => {
  const params = {
    username,
    email,
    password,
    first_name,
    last_name,
  };

  try {
    const res = await axios.post(endpoints.auth.signUp, params);

    const { token } = res.data; // Go backend returns 'token' field

    if (!token) {
      throw new Error('Access token not found in response');
    }

    setSession(token);
  } catch (error) {
    console.error('Error during sign up:', error);
    throw error;
  }
};

/** **************************************
 * Sign out
 *************************************** */
export const signOut = async (): Promise<void> => {
  try {
    await setSession(null);
  } catch (error) {
    console.error('Error during sign out:', error);
    throw error;
  }
};
