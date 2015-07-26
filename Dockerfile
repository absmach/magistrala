###
# Mainflux Dockerfile
###
# Set the base image to Node, onbuild variant: https://registry.hub.docker.com/_/node/
FROM node:0.10-onbuild

# Maintained by Mainflux team
MAINTAINER Mainflux <docker@mainflux.com>

# Log info
RUN echo "Starting Mainflux server..."

###
# Installations
###
# Add Gulp globally
RUN npm install -g gulp

# Gulp also demands to be saved locally
RUN npm install --save-dev gulp

# Finally, install all project Node modules
RUN npm install

###
# Setup the port
###
# Run Mainflux on port 80
ENV PORT 80

# Expose port on which we run Mainflux
EXPOSE $PORT

###
# Run main command from entrypoint and parameters in CMD[]
###
# Default port to execute the entrypoint (MongoDB)
CMD [""]

# Set default container command
ENTRYPOINT gulp

