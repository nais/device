<?php declare(strict_types=1);
namespace Nais\Device\Approval\Session;

class User {
    private string $objectId;
    private string $name;

    /**
     * Class constructor
     *
     * @param string $objectId
     * @param string $name
     */
    public function __construct(string $objectId, string $name) {
        $this->objectId = $objectId;
        $this->name     = $name;
    }

    /**
     * Get the object ID
     *
     * @return string
     */
    public function getObjectId() : string {
        return $this->objectId;
    }

    /**
     * Get the name property
     *
     * @return string
     */
    public function getName() : string {
        return $this->name;
    }
}