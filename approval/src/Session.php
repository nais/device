<?php declare(strict_types=1);
namespace Nais\Device\Approval;

use Nais\Device\Approval\Session\User;
use RuntimeException;

class Session {
    /**
     * Start the session
     *
     * @codeCoverageIgnore
     * @return self
     */
    public function start() : self {
        session_start();
        return $this;
    }

    /**
     * Set a user object
     *
     * @param User $user
     * @throws RuntimeException
     * @return void
     */
    public function setUser(User $user) : void {
        $_SESSION['user'] = $user;
    }

    /**
     * Get the user instace
     *
     * @return ?User
     */
    public function getUser() : ?User {
        /** @var ?User */
        $user = $_SESSION['user'] ?? null;

        if (null === $user || !$user instanceof User) {
            $_SESSION['user'] = null;
            return null;
        }

        return $user;
    }

    /**
     * Check if a user exists in the session
     *
     * @return bool
     */
    public function hasUser() : bool {
        return array_key_exists('user', $_SESSION) && $_SESSION['user'] instanceof User;
    }

    /**
     * Remove the current user
     *
     * @return void
     */
    public function deleteUser() : void {
        unset($_SESSION['user']);
    }

    /**
     * Destroy the current session
     *
     * @codeCoverageIgnore
     * @return self
     */
    public function destroy() : self {
        session_destroy();
        return $this;
    }
}