<?php declare(strict_types=1);
namespace Nais\Device\Approval\Session;

use PHPUnit\Framework\TestCase;

/**
 * @coversDefaultClass Nais\Device\Approval\Session\User
 */
class UserTest extends TestCase {
    /**
     * @covers ::__construct
     * @covers ::getObjectId
     * @covers ::getName
     */
    public function testCanGetValues() : void {
        $user = new User('id', 'name');
        $this->assertSame('id', $user->getObjectId());
        $this->assertSame('name', $user->getName());
    }
}